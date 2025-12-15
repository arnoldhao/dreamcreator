package appmenu

import (
	"strings"

	"dreamcreator/backend/consts"
	"dreamcreator/backend/services/preferences"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const localeChangedEvent = "app:localeChanged"

type Controller struct {
	app  *application.App
	menu *application.Menu

	isMacOS bool
	pref    *preferences.Service
	loc     *Localizer

	appName      string
	openSettings func()

	helpURLZh string
	helpURLEn string

	appMenuRoot    *application.MenuItem
	fileMenuRoot   *application.MenuItem
	editMenuRoot   *application.MenuItem
	windowMenuRoot *application.MenuItem
	helpMenuRoot   *application.MenuItem

	aboutItem    *application.MenuItem
	settingsItem *application.MenuItem
	hideItem     *application.MenuItem
	showAllItem  *application.MenuItem
	quitItem     *application.MenuItem

	editUndoItem          *application.MenuItem
	editRedoItem          *application.MenuItem
	editCutItem           *application.MenuItem
	editCopyItem          *application.MenuItem
	editPasteItem         *application.MenuItem
	editPasteMatchItem    *application.MenuItem
	editDeleteItem        *application.MenuItem
	editSelectAllItem     *application.MenuItem
	editSpeechMenuItem    *application.MenuItem
	editStartSpeakingItem *application.MenuItem
	editStopSpeakingItem  *application.MenuItem

	windowMinimiseItem        *application.MenuItem
	windowZoomItem            *application.MenuItem
	windowBringAllToFrontItem *application.MenuItem
	windowCloseItem           *application.MenuItem

	helpLearnMoreItem *application.MenuItem
}

type Options struct {
	IsMacOS       bool
	OpenSettings  func()
	Preferences   *preferences.Service
	InitialLocale string
	AppName       string
	HelpURLZh     string
	HelpURLEn     string
}

func Install(app *application.App, menu *application.Menu, opts Options) *Controller {
	c := &Controller{
		app:          app,
		menu:         menu,
		isMacOS:      opts.IsMacOS,
		pref:         opts.Preferences,
		openSettings: opts.OpenSettings,
		loc:          NewLocalizer(opts.InitialLocale),
		appName:      strings.TrimSpace(opts.AppName),
		helpURLZh:    strings.TrimSpace(opts.HelpURLZh),
		helpURLEn:    strings.TrimSpace(opts.HelpURLEn),
	}
	if c.helpURLZh == "" {
		c.helpURLZh = consts.HELP_URL_ZH
	}
	if c.helpURLEn == "" {
		c.helpURLEn = consts.HELP_URL_EN
	}

	if c.pref != nil && strings.TrimSpace(opts.InitialLocale) == "" {
		c.loc.SetLocale(c.pref.GetLanguage())
	}

	c.build()
	c.updateLabels()

	c.app.Event.On(localeChangedEvent, func(evt *application.CustomEvent) {
		next := ""
		if evt != nil {
			if s, ok := evt.Data.(string); ok {
				next = s
			}
		}
		if strings.TrimSpace(next) == "" && c.pref != nil {
			next = c.pref.GetLanguage()
		}
		c.loc.SetLocale(next)
		c.updateLabels()
	})

	return c
}

func (c *Controller) build() {
	if c.isMacOS {
		c.buildMacMenus()
		return
	}
	c.buildNonMacMenus()
}

func (c *Controller) buildMacMenus() {
	c.buildMacAppMenu()
	c.buildMacEditMenu()
	c.buildMacWindowMenu()
	c.buildMacHelpMenu()
}

func (c *Controller) buildMacAppMenu() {
	if c.menu == nil {
		return
	}

	// Put Settings as the second item in the app menu:
	// About <AppName>
	// -----------
	// Settings...
	// ...
	c.appMenuRoot = c.menu.FindByRole(application.AppMenu)
	if c.appMenuRoot == nil || c.appMenuRoot.GetSubmenu() == nil {
		// Fallback to top-level.
		c.settingsItem = c.menu.Add(c.loc.T("menu.app.settings")).OnClick(func(_ *application.Context) { c.openSettingsSafe() }).SetAccelerator("CmdOrCtrl+,")
		return
	}

	sub := c.appMenuRoot.GetSubmenu()
	sub.Clear()
	sub.AddRole(application.About)
	c.aboutItem = sub.FindByRole(application.About)
	sub.AddSeparator()
	c.settingsItem = sub.Add(c.loc.T("menu.app.settings")).OnClick(func(_ *application.Context) { c.openSettingsSafe() }).SetAccelerator("CmdOrCtrl+,")
	sub.AddSeparator()
	sub.AddRole(application.Hide)
	sub.AddRole(application.ShowAll)
	c.hideItem = sub.FindByRole(application.Hide)
	c.showAllItem = sub.FindByRole(application.ShowAll)
	sub.AddSeparator()
	sub.AddRole(application.Quit)
	c.quitItem = sub.FindByRole(application.Quit)
}

func (c *Controller) buildMacEditMenu() {
	if c.menu == nil {
		return
	}
	c.editMenuRoot = c.menu.FindByRole(application.EditMenu)
	if c.editMenuRoot == nil || c.editMenuRoot.GetSubmenu() == nil {
		return
	}
	sub := c.editMenuRoot.GetSubmenu()
	sub.Clear()

	sub.AddRole(application.Undo)
	sub.AddRole(application.Redo)
	sub.AddSeparator()
	sub.AddRole(application.Cut)
	sub.AddRole(application.Copy)
	sub.AddRole(application.Paste)
	sub.AddRole(application.PasteAndMatchStyle)
	sub.AddRole(application.Delete)
	sub.AddRole(application.SelectAll)

	sub.AddSeparator()
	speechLabel := c.loc.T("menu.edit.speech")
	speech := sub.AddSubmenu(speechLabel)
	speech.AddRole(application.StartSpeaking)
	speech.AddRole(application.StopSpeaking)
	c.editSpeechMenuItem = sub.FindByLabel(speechLabel)

	c.editUndoItem = sub.FindByRole(application.Undo)
	c.editRedoItem = sub.FindByRole(application.Redo)
	c.editCutItem = sub.FindByRole(application.Cut)
	c.editCopyItem = sub.FindByRole(application.Copy)
	c.editPasteItem = sub.FindByRole(application.Paste)
	c.editPasteMatchItem = sub.FindByRole(application.PasteAndMatchStyle)
	c.editDeleteItem = sub.FindByRole(application.Delete)
	c.editSelectAllItem = sub.FindByRole(application.SelectAll)
	c.editStartSpeakingItem = speech.FindByRole(application.StartSpeaking)
	c.editStopSpeakingItem = speech.FindByRole(application.StopSpeaking)
}

func (c *Controller) buildMacWindowMenu() {
	if c.menu == nil {
		return
	}
	c.windowMenuRoot = c.menu.FindByRole(application.WindowMenu)
	if c.windowMenuRoot == nil || c.windowMenuRoot.GetSubmenu() == nil {
		return
	}
	sub := c.windowMenuRoot.GetSubmenu()
	sub.Clear()
	sub.AddRole(application.Minimise)
	sub.AddRole(application.Zoom)
	sub.AddSeparator()
	sub.AddRole(application.BringAllToFront)

	c.windowMinimiseItem = sub.FindByRole(application.Minimise)
	c.windowZoomItem = sub.FindByRole(application.Zoom)
	c.windowBringAllToFrontItem = sub.FindByRole(application.BringAllToFront)
}

func (c *Controller) buildMacHelpMenu() {
	if c.menu == nil {
		return
	}
	c.helpMenuRoot = c.menu.FindByRole(application.HelpMenu)
	if c.helpMenuRoot == nil || c.helpMenuRoot.GetSubmenu() == nil {
		return
	}
	sub := c.helpMenuRoot.GetSubmenu()
	sub.Clear()
	c.helpLearnMoreItem = sub.Add(c.loc.T("menu.help.learn_more")).OnClick(func(_ *application.Context) {
		_ = c.app.Browser.OpenURL(c.helpURLForLocale())
	})
}

func (c *Controller) buildNonMacMenus() {
	c.buildNonMacFileMenu()
	c.buildNonMacEditMenu()
	c.buildNonMacWindowMenu()
	c.buildNonMacHelpMenu()
}

func (c *Controller) buildNonMacFileMenu() {
	if c.menu == nil {
		return
	}
	c.fileMenuRoot = c.menu.FindByRole(application.FileMenu)
	if c.fileMenuRoot == nil || c.fileMenuRoot.GetSubmenu() == nil {
		return
	}
	sub := c.fileMenuRoot.GetSubmenu()
	sub.Clear()
	c.settingsItem = sub.Add(c.loc.T("menu.app.settings")).OnClick(func(_ *application.Context) { c.openSettingsSafe() }).SetAccelerator("CmdOrCtrl+,")
	sub.AddSeparator()
	sub.AddRole(application.Quit)
	c.quitItem = sub.FindByRole(application.Quit)
}

func (c *Controller) buildNonMacEditMenu() {
	if c.menu == nil {
		return
	}
	c.editMenuRoot = c.menu.FindByRole(application.EditMenu)
	if c.editMenuRoot == nil || c.editMenuRoot.GetSubmenu() == nil {
		return
	}
	sub := c.editMenuRoot.GetSubmenu()
	sub.Clear()
	sub.AddRole(application.Undo)
	sub.AddRole(application.Redo)
	sub.AddSeparator()
	sub.AddRole(application.Cut)
	sub.AddRole(application.Copy)
	sub.AddRole(application.Paste)
	sub.AddSeparator()
	sub.AddRole(application.Delete)
	sub.AddSeparator()
	sub.AddRole(application.SelectAll)

	c.editUndoItem = sub.FindByRole(application.Undo)
	c.editRedoItem = sub.FindByRole(application.Redo)
	c.editCutItem = sub.FindByRole(application.Cut)
	c.editCopyItem = sub.FindByRole(application.Copy)
	c.editPasteItem = sub.FindByRole(application.Paste)
	c.editDeleteItem = sub.FindByRole(application.Delete)
	c.editSelectAllItem = sub.FindByRole(application.SelectAll)
}

func (c *Controller) buildNonMacWindowMenu() {
	if c.menu == nil {
		return
	}
	c.windowMenuRoot = c.menu.FindByRole(application.WindowMenu)
	if c.windowMenuRoot == nil || c.windowMenuRoot.GetSubmenu() == nil {
		return
	}
	sub := c.windowMenuRoot.GetSubmenu()
	sub.Clear()
	sub.AddRole(application.CloseWindow)
	c.windowCloseItem = sub.FindByRole(application.CloseWindow)
}

func (c *Controller) buildNonMacHelpMenu() {
	if c.menu == nil {
		return
	}
	c.helpMenuRoot = c.menu.FindByRole(application.HelpMenu)
	if c.helpMenuRoot == nil || c.helpMenuRoot.GetSubmenu() == nil {
		return
	}
	sub := c.helpMenuRoot.GetSubmenu()
	sub.Clear()
	c.helpLearnMoreItem = sub.Add(c.loc.T("menu.help.learn_more")).OnClick(func(_ *application.Context) {
		_ = c.app.Browser.OpenURL(c.helpURLForLocale())
	})
}

func (c *Controller) updateLabels() {
	vars := map[string]string{"app": c.appName}

	if c.fileMenuRoot != nil {
		c.fileMenuRoot.SetLabel(c.loc.T("menu.file.title"))
	}
	if c.editMenuRoot != nil {
		c.editMenuRoot.SetLabel(c.loc.T("menu.edit.title"))
	}
	if c.windowMenuRoot != nil {
		c.windowMenuRoot.SetLabel(c.loc.T("menu.window.title"))
	}
	if c.helpMenuRoot != nil {
		c.helpMenuRoot.SetLabel(c.loc.T("menu.help.title"))
	}

	if c.aboutItem != nil {
		c.aboutItem.SetLabel(c.loc.Format("menu.app.about", vars))
	}
	if c.settingsItem != nil {
		c.settingsItem.SetLabel(c.loc.T("menu.app.settings"))
	}
	if c.hideItem != nil {
		c.hideItem.SetLabel(c.loc.Format("menu.app.hide", vars))
	}
	if c.showAllItem != nil {
		c.showAllItem.SetLabel(c.loc.T("menu.app.show_all"))
	}
	if c.quitItem != nil {
		c.quitItem.SetLabel(c.loc.Format("menu.app.quit", vars))
	}

	if c.editUndoItem != nil {
		c.editUndoItem.SetLabel(c.loc.T("menu.edit.undo"))
	}
	if c.editRedoItem != nil {
		c.editRedoItem.SetLabel(c.loc.T("menu.edit.redo"))
	}
	if c.editCutItem != nil {
		c.editCutItem.SetLabel(c.loc.T("menu.edit.cut"))
	}
	if c.editCopyItem != nil {
		c.editCopyItem.SetLabel(c.loc.T("menu.edit.copy"))
	}
	if c.editPasteItem != nil {
		c.editPasteItem.SetLabel(c.loc.T("menu.edit.paste"))
	}
	if c.editPasteMatchItem != nil {
		c.editPasteMatchItem.SetLabel(c.loc.T("menu.edit.paste_and_match_style"))
	}
	if c.editDeleteItem != nil {
		c.editDeleteItem.SetLabel(c.loc.T("menu.edit.delete"))
	}
	if c.editSelectAllItem != nil {
		c.editSelectAllItem.SetLabel(c.loc.T("menu.edit.select_all"))
	}
	if c.editSpeechMenuItem != nil {
		c.editSpeechMenuItem.SetLabel(c.loc.T("menu.edit.speech"))
	}
	if c.editStartSpeakingItem != nil {
		c.editStartSpeakingItem.SetLabel(c.loc.T("menu.edit.start_speaking"))
	}
	if c.editStopSpeakingItem != nil {
		c.editStopSpeakingItem.SetLabel(c.loc.T("menu.edit.stop_speaking"))
	}

	if c.windowMinimiseItem != nil {
		c.windowMinimiseItem.SetLabel(c.loc.T("menu.window.minimise"))
	}
	if c.windowZoomItem != nil {
		c.windowZoomItem.SetLabel(c.loc.T("menu.window.zoom"))
	}
	if c.windowBringAllToFrontItem != nil {
		c.windowBringAllToFrontItem.SetLabel(c.loc.T("menu.window.bring_all_to_front"))
	}
	if c.windowCloseItem != nil {
		c.windowCloseItem.SetLabel(c.loc.T("menu.window.close"))
	}

	if c.helpLearnMoreItem != nil {
		c.helpLearnMoreItem.SetLabel(c.loc.T("menu.help.learn_more"))
	}
}

func (c *Controller) openSettingsSafe() {
	if c.openSettings != nil {
		c.openSettings()
	}
}

func (c *Controller) helpURLForLocale() string {
	locale := strings.ToLower(c.loc.Locale())
	if strings.HasPrefix(locale, "zh") {
		return c.helpURLZh
	}
	return c.helpURLEn
}
