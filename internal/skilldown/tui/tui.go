// Package tui 提供交互式 registry 浏览、skill 预览和安装入口。
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/finger/skill-down/internal/skilldown/app"
	"github.com/finger/skill-down/internal/skilldown/config"
)

const registryLoadDelay = 250 * time.Millisecond

type pane int

const (
	paneRegistry pane = iota
	paneSkills
	panePreview
)

type inputMode int

const (
	modeNormal inputMode = iota
	modeRegistrySearch
	modeSkillSearch
	modeAddRegistry
	modeSkillDir
	modeInstallConfirm
)

// model 保存 TUI 会话的交互状态。
type model struct {
	// ctx 是 TUI 生命周期内传递给 app 用例的上下文。
	ctx context.Context
	// svc 是 registry、搜索和安装动作的 app 用例入口。
	svc app.Service
	// initialRepo 是命令行显式传入的初始仓库地址。
	initialRepo string
	// focus 表示当前接收 Vim 快捷键的栏目。
	focus pane
	// mode 表示当前是否处于搜索、输入目录或安装确认。
	mode inputMode
	// repositories 是左栏展示的完整 registry 仓库列表。
	repositories []config.Repository
	// filteredRepositories 是左栏按搜索词过滤后的列表。
	filteredRepositories []config.Repository
	// registryCursor 是左栏当前光标位置。
	registryCursor int
	// skills 是当前仓库搜索得到的完整 skill 列表。
	skills []app.SkillResult
	// filteredSkills 是中栏按搜索词过滤后的列表。
	filteredSkills []app.SkillResult
	// skillCursor 是中栏当前光标位置。
	skillCursor int
	// selected 保存中栏已勾选的 skill 名称。
	selected map[string]bool
	// skillDirPath 是当前搜索和安装覆盖使用的 skill 目录。
	skillDirPath string
	// targetRoot 是安装确认框中的目标根目录。
	targetRoot string
	// force 表示安装时是否覆盖已有目录。
	force bool
	// input 是搜索、添加 registry、设置目录和安装确认共用文本输入。
	input textinput.Model
	// status 是底部状态栏展示的最近动作结果。
	status string
	// err 是最近一次业务动作错误。
	err error
	// width 是当前终端宽度。
	width int
	// height 是当前终端高度。
	height int
	// registryLoadToken 用于忽略过期的 registry 防抖加载消息。
	registryLoadToken int
	// currentRepo 是当前中栏 skill 来源仓库。
	currentRepo config.Repository
	// showHelp 控制是否展示快捷键帮助视图。
	showHelp bool
}

type repositoriesLoadedMsg struct {
	// repositories 是从配置中读取到的 registry 仓库列表。
	repositories []config.Repository
	// err 是加载 registry 时产生的错误。
	err error
}

type registryLoadDueMsg struct {
	// token 是本次防抖加载的版本号。
	token int
	// repo 是防抖结束后应搜索的仓库。
	repo config.Repository
}

type searchResultMsg struct {
	// result 是 app 搜索返回的结构化 skill 列表。
	result app.SearchResult
	// err 是搜索过程中产生的错误。
	err error
}

type installResultMsg struct {
	// skillName 是本次安装动作对应的 skill 名称。
	skillName string
	// result 是 app 安装返回的结构化结果。
	result app.InstallResult
	// err 是安装过程中产生的错误。
	err error
}

// Run 启动 Bubble Tea 交互式 TUI。
func Run(ctx context.Context, svc app.Service, initialRepo string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	program := tea.NewProgram(newModel(ctx, svc, initialRepo), tea.WithContext(ctx), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

// newModel 创建 TUI 初始状态。
func newModel(ctx context.Context, svc app.Service, initialRepo string) model {
	input := textinput.New()
	input.Prompt = "> "
	input.CharLimit = 512
	input.Width = 48

	return model{
		ctx:         ctx,
		svc:         svc,
		initialRepo: strings.TrimSpace(initialRepo),
		focus:       paneRegistry,
		mode:        modeNormal,
		selected:    map[string]bool{},
		input:       input,
		width:       100,
		height:      30,
		status:      "ready",
	}
}

// Init 初始化 registry 列表或显式仓库搜索。
func (m model) Init() tea.Cmd {
	if m.initialRepo != "" {
		return m.searchCommand(config.Repository{Name: m.initialRepo, URL: m.initialRepo})
	}
	return m.listRepositoriesCommand()
}

// Update 根据 Bubble Tea 消息更新 TUI 状态。
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case repositoriesLoadedMsg:
		return m.handleRepositoriesLoaded(msg)
	case registryLoadDueMsg:
		return m.handleRegistryLoadDue(msg)
	case searchResultMsg:
		return m.handleSearchResult(msg)
	case installResultMsg:
		return m.handleInstallResult(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	default:
		return m, nil
	}
}

// View 渲染三栏主界面和底部状态。
func (m model) View() string {
	if m.showHelp {
		return m.helpView()
	}

	leftWidth := max(20, m.width/4)
	middleWidth := max(28, m.width/3)
	rightWidth := max(30, m.width-leftWidth-middleWidth-4)

	left := m.paneView("Registry", m.registryLines(), leftWidth, m.focus == paneRegistry)
	middle := m.paneView("Skills", m.skillLines(), middleWidth, m.focus == paneSkills)
	right := m.paneView("Preview", m.previewLines(), rightWidth, m.focus == panePreview)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, middle, right)

	status := m.status
	if m.err != nil {
		status = m.err.Error()
	}
	if m.mode != modeNormal {
		status = m.prompt() + m.input.View()
	}
	return body + "\n" + status
}

// handleRepositoriesLoaded 将 registry 加载结果写入左栏状态。
func (m model) handleRepositoriesLoaded(msg repositoriesLoadedMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.status = "加载 registry 失败"
		return m, nil
	}
	m.repositories = msg.repositories
	m.filteredRepositories = msg.repositories
	m.registryCursor = clampCursor(m.registryCursor, len(m.filteredRepositories))
	m.status = fmt.Sprintf("已加载 %d 个 registry", len(m.repositories))
	if len(m.filteredRepositories) == 0 {
		return m, nil
	}
	return m.scheduleRegistryLoad()
}

// handleRegistryLoadDue 在防抖消息仍然有效时触发仓库搜索。
func (m model) handleRegistryLoadDue(msg registryLoadDueMsg) (model, tea.Cmd) {
	if msg.token != m.registryLoadToken {
		return m, nil
	}
	return m, m.searchCommand(msg.repo)
}

// handleSearchResult 将搜索结果写入中栏 skill 列表和右栏预览来源。
func (m model) handleSearchResult(msg searchResultMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.status = "搜索失败"
		return m, nil
	}
	m.err = nil
	m.skills = flattenSkills(msg.result)
	m.filteredSkills = m.skills
	m.skillCursor = clampCursor(m.skillCursor, len(m.filteredSkills))
	if len(msg.result.Repositories) > 0 {
		m.currentRepo = msg.result.Repositories[0].Repository
	}
	m.status = fmt.Sprintf("已发现 %d 个 skill", len(m.skills))
	return m, nil
}

// handleInstallResult 更新安装动作的底部状态。
func (m model) handleInstallResult(msg installResultMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.status = fmt.Sprintf("%s 安装失败", msg.skillName)
		return m, nil
	}
	m.err = nil
	m.status = fmt.Sprintf("%s 安装完成", msg.skillName)
	return m, nil
}

// handleKey 处理普通模式下的 Vim 风格快捷键。
func (m model) handleKey(msg tea.KeyMsg) (model, tea.Cmd) {
	if m.mode != modeNormal {
		return m.handleInputKey(msg)
	}
	if m.showHelp && msg.String() != "?" {
		m.showHelp = false
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "h":
		m.focus = pane(max(int(paneRegistry), int(m.focus)-1))
		return m, nil
	case "l":
		m.focus = pane(min(int(panePreview), int(m.focus)+1))
		return m, nil
	case "H":
		m.focus = paneRegistry
		return m, nil
	case "L":
		m.focus = panePreview
		return m, nil
	case "j":
		return m.moveCursor(1)
	case "k":
		return m.moveCursor(-1)
	case "/":
		return m.startSearch()
	case "a":
		return m.startAddOrSkillDir()
	case " ":
		return m.toggleSelected(), nil
	case "r":
		return m.refreshCurrentRepo()
	case "i":
		return m.openInstallConfirm()
	default:
		return m, nil
	}
}

// handleInputKey 处理搜索、目录输入和安装确认模式的按键。
func (m model) handleInputKey(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.input.Blur()
		return m, nil
	case "enter":
		return m.commitInput()
	case "f":
		if m.mode == modeInstallConfirm {
			m.force = !m.force
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// moveCursor 在当前聚焦栏目内移动光标。
func (m model) moveCursor(delta int) (model, tea.Cmd) {
	switch m.focus {
	case paneRegistry:
		repositories := m.visibleRepositories()
		m.registryCursor = clampCursor(m.registryCursor+delta, len(repositories))
		return m.scheduleRegistryLoad()
	case paneSkills:
		m.skillCursor = clampCursor(m.skillCursor+delta, len(m.visibleSkills()))
		return m, nil
	default:
		return m, nil
	}
}

// startSearch 进入当前栏目对应的搜索输入模式。
func (m model) startSearch() (model, tea.Cmd) {
	if m.focus == paneRegistry {
		m.mode = modeRegistrySearch
		m.input.Placeholder = "registry"
	} else {
		m.mode = modeSkillSearch
		m.input.Placeholder = "skill"
	}
	m.input.SetValue("")
	return m.focusInput()
}

// startAddOrSkillDir 根据当前栏目进入添加 registry 或设置 skillDirPath 模式。
func (m model) startAddOrSkillDir() (model, tea.Cmd) {
	if m.focus == paneRegistry {
		m.mode = modeAddRegistry
		m.input.Placeholder = "repo url"
	} else if m.focus == paneSkills {
		m.mode = modeSkillDir
		m.input.Placeholder = "skill dir"
		m.input.SetValue(m.skillDirPath)
		return m.focusInput()
	} else {
		return m, nil
	}
	m.input.SetValue("")
	return m.focusInput()
}

// openInstallConfirm 在已有选中 skill 时打开安装确认输入。
func (m model) openInstallConfirm() (model, tea.Cmd) {
	if selectedNames(m.selected, m.visibleSkills()) == nil {
		m.status = "请先选择 skill"
		return m, nil
	}
	m.mode = modeInstallConfirm
	m.input.Placeholder = ".codex/skills"
	m.input.SetValue(m.targetRoot)
	return m.focusInput()
}

// commitInput 根据输入模式提交文本框中的值。
func (m model) commitInput() (model, tea.Cmd) {
	mode := m.mode
	value := strings.TrimSpace(m.input.Value())
	m.mode = modeNormal
	m.input.Blur()

	switch mode {
	case modeRegistrySearch:
		m.filteredRepositories = filterRepositories(m.repositories, value)
		m.registryCursor = clampCursor(0, len(m.filteredRepositories))
		return m.scheduleRegistryLoad()
	case modeSkillSearch:
		m.filteredSkills = filterSkills(m.skills, value)
		m.skillCursor = clampCursor(0, len(m.filteredSkills))
		return m, nil
	case modeAddRegistry:
		if value == "" {
			m.status = "registry 地址不能为空"
			return m, nil
		}
		return m, m.addRegistryCommand(value)
	case modeSkillDir:
		m.skillDirPath = value
		return m.refreshCurrentRepo()
	case modeInstallConfirm:
		m.targetRoot = value
		return m, m.installSelectedCommand()
	default:
		return m, nil
	}
}

// focusInput 聚焦共用文本输入框。
func (m model) focusInput() (model, tea.Cmd) {
	cmd := m.input.Focus()
	return m, cmd
}

// refreshCurrentRepo 使用当前仓库和 skillDirPath 重新搜索 skill。
func (m model) refreshCurrentRepo() (model, tea.Cmd) {
	repository, ok := m.activeRepository()
	if !ok {
		m.status = "没有可刷新的仓库"
		return m, nil
	}
	return m, m.searchCommand(repository)
}

// toggleSelected 切换当前 skill 的选中状态。
func (m model) toggleSelected() model {
	skills := m.visibleSkills()
	if m.focus != paneSkills || len(skills) == 0 {
		return m
	}
	name := skills[m.skillCursor].Name
	if m.selected[name] {
		delete(m.selected, name)
	} else {
		m.selected[name] = true
	}
	m.status = fmt.Sprintf("已选择 %d 个 skill", len(m.selected))
	return m
}

// scheduleRegistryLoad 创建 registry 光标停留后的防抖加载命令。
func (m model) scheduleRegistryLoad() (model, tea.Cmd) {
	repositories := m.visibleRepositories()
	if len(repositories) == 0 {
		return m, nil
	}
	m.registryLoadToken++
	token := m.registryLoadToken
	repository := repositories[m.registryCursor]
	return m, tea.Tick(registryLoadDelay, func(time.Time) tea.Msg {
		return registryLoadDueMsg{token: token, repo: repository}
	})
}

// listRepositoriesCommand 创建读取 registry 列表的异步命令。
func (m model) listRepositoriesCommand() tea.Cmd {
	return func() tea.Msg {
		repositories, err := m.svc.ListRepositories(m.ctx)
		return repositoriesLoadedMsg{repositories: repositories, err: err}
	}
}

// searchCommand 创建搜索指定仓库 skill 的异步命令。
func (m model) searchCommand(repository config.Repository) tea.Cmd {
	return func() tea.Msg {
		result, err := m.svc.Search(m.ctx, app.SearchRequest{
			RepoURL:      repository.URL,
			SkillDirPath: m.requestSkillDir(repository),
		})
		return searchResultMsg{result: result, err: err}
	}
}

// addRegistryCommand 创建添加 registry 并重新加载列表的异步命令。
func (m model) addRegistryCommand(repoURL string) tea.Cmd {
	return func() tea.Msg {
		err := m.svc.AddRepository(m.ctx, app.AddRepositoryRequest{
			Name: repoURL,
			URL:  repoURL,
		})
		if err != nil {
			return repositoriesLoadedMsg{err: err}
		}
		repositories, err := m.svc.ListRepositories(m.ctx)
		return repositoriesLoadedMsg{repositories: repositories, err: err}
	}
}

// installSelectedCommand 为每个已选 skill 创建安装命令。
func (m model) installSelectedCommand() tea.Cmd {
	names := selectedNames(m.selected, m.visibleSkills())
	repository, ok := m.activeRepository()
	if !ok || len(names) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(names))
	for _, name := range names {
		skillName := name
		cmds = append(cmds, func() tea.Msg {
			result, err := m.svc.Install(m.ctx, app.InstallRequest{
				RepoURL:      repository.URL,
				SkillName:    skillName,
				SkillDirPath: m.requestSkillDir(repository),
				TargetRoot:   m.targetRoot,
				Force:        m.force,
			})
			return installResultMsg{skillName: skillName, result: result, err: err}
		})
	}
	return tea.Batch(cmds...)
}

// activeRepository 返回当前搜索和安装应使用的仓库。
func (m model) activeRepository() (config.Repository, bool) {
	if m.currentRepo.URL != "" {
		return m.currentRepo, true
	}
	if len(m.filteredRepositories) == 0 {
		repositories := m.visibleRepositories()
		if len(repositories) == 0 {
			return config.Repository{}, false
		}
		return repositories[m.registryCursor], true
	}
	return m.filteredRepositories[m.registryCursor], true
}

// requestSkillDir 返回本次 app 请求应使用的 skill 目录。
func (m model) requestSkillDir(repository config.Repository) string {
	if strings.TrimSpace(m.skillDirPath) != "" {
		return m.skillDirPath
	}
	return repository.SkillDir
}

// visibleRepositories 返回当前左栏实际显示的仓库列表。
func (m model) visibleRepositories() []config.Repository {
	if m.filteredRepositories != nil {
		return m.filteredRepositories
	}
	return m.repositories
}

// visibleSkills 返回当前中栏实际显示的 skill 列表。
func (m model) visibleSkills() []app.SkillResult {
	if m.filteredSkills != nil {
		return m.filteredSkills
	}
	return m.skills
}

// paneView 渲染单个带边框的栏目。
func (m model) paneView(title string, lines []string, width int, focused bool) string {
	style := lipgloss.NewStyle().Width(width).Height(max(8, m.height-3)).Padding(0, 1).Border(lipgloss.NormalBorder())
	if focused {
		style = style.BorderForeground(lipgloss.Color("62"))
	}
	content := title + "\n" + strings.Join(lines, "\n")
	return style.Render(content)
}

// registryLines 生成左栏 registry 文本行。
func (m model) registryLines() []string {
	repositories := m.visibleRepositories()
	if len(repositories) == 0 {
		return []string{"暂无 registry"}
	}
	lines := make([]string, 0, len(repositories))
	for i, repository := range repositories {
		cursor := " "
		if i == m.registryCursor {
			cursor = ">"
		}
		lines = append(lines, fmt.Sprintf("%s %s", cursor, repository.Name))
	}
	return lines
}

// skillLines 生成中栏 skill 文本行。
func (m model) skillLines() []string {
	skills := m.visibleSkills()
	if len(skills) == 0 {
		return []string{"暂无 skill"}
	}
	lines := make([]string, 0, len(skills))
	for i, skill := range skills {
		cursor := " "
		if i == m.skillCursor {
			cursor = ">"
		}
		checked := " "
		if m.selected[skill.Name] {
			checked = "x"
		}
		lines = append(lines, fmt.Sprintf("%s [%s] %s", cursor, checked, skill.Name))
	}
	return lines
}

// previewLines 生成右栏 SKILL.md 预览文本行。
func (m model) previewLines() []string {
	skills := m.visibleSkills()
	if len(skills) == 0 {
		return []string{"移动到 skill 后预览 SKILL.md"}
	}
	skill := skills[m.skillCursor]
	lines := []string{skill.Name, skill.Description, ""}
	lines = append(lines, strings.Split(skill.Content, "\n")...)
	return lines
}

// helpView 渲染快捷键帮助。
func (m model) helpView() string {
	return strings.Join([]string{
		"h/l 切换栏目   j/k 移动光标   H/L 跳到首尾栏目",
		"/ 搜索   a 添加 registry 或设置 skillDirPath",
		"space 选择 skill   r 刷新   i 安装确认",
		"安装确认中 f 切换覆盖，enter 确认，esc 取消",
		"? 关闭帮助   q 退出",
	}, "\n")
}

// prompt 返回当前输入模式的提示文本。
func (m model) prompt() string {
	switch m.mode {
	case modeRegistrySearch:
		return "搜索 registry "
	case modeSkillSearch:
		return "搜索 skill "
	case modeAddRegistry:
		return "添加 registry "
	case modeSkillDir:
		return "skillDirPath "
	case modeInstallConfirm:
		return fmt.Sprintf("安装到 force=%t ", m.force)
	default:
		return ""
	}
}

func flattenSkills(result app.SearchResult) []app.SkillResult {
	var skills []app.SkillResult
	for _, group := range result.Repositories {
		skills = append(skills, group.Skills...)
	}
	return skills
}

func filterRepositories(repositories []config.Repository, query string) []config.Repository {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return repositories
	}
	var filtered []config.Repository
	for _, repository := range repositories {
		haystack := strings.ToLower(repository.Name + " " + repository.URL)
		if strings.Contains(haystack, query) {
			filtered = append(filtered, repository)
		}
	}
	return filtered
}

func filterSkills(skills []app.SkillResult, query string) []app.SkillResult {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return skills
	}
	var filtered []app.SkillResult
	for _, skill := range skills {
		haystack := strings.ToLower(skill.Name + " " + skill.Description + " " + skill.Path)
		if strings.Contains(haystack, query) {
			filtered = append(filtered, skill)
		}
	}
	return filtered
}

func selectedNames(selected map[string]bool, skills []app.SkillResult) []string {
	var names []string
	for _, skill := range skills {
		if selected[skill.Name] {
			names = append(names, skill.Name)
		}
	}
	return names
}

func clampCursor(cursor int, length int) int {
	if length == 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= length {
		return length - 1
	}
	return cursor
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
