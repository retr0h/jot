local M = {}

local defaults = {
  keys = {
    jump      = '<leader>jj',
    search    = '<leader>js',
    tags      = '<leader>jt',
    backlinks = '<leader>jb',
    links     = '<leader>jl',
    new       = '<leader>jn',
    done      = '<leader>jx',
    follow    = '<leader>jf',
    help      = '<leader>j?',
  },
}

function M.setup(opts)
  opts = vim.tbl_deep_extend('force', defaults, opts or {})
  local notes_dir = vim.env.JOT_NOTES_DIR
  if not notes_dir or notes_dir == '' then return end

  -- Setup obsidian.nvim with workspace
  local obs_ok, obsidian = pcall(require, 'obsidian')
  if obs_ok then
    obsidian.setup({
      workspaces = {{ name = 'jot', path = notes_dir }},
      ui = {
        enable = false,
        checkboxes = {
          [' '] = { char = ' ', order = 1 },
          ['x'] = { char = 'x', order = 2 },
        },
      },
      completion = {
        nvim_cmp = true,
        min_chars = 2,
        prepend_note_id = false,
      },
    })
  end

  -- Setup markview.nvim
  local mv_ok, markview = pcall(require, 'markview')
  if mv_ok then markview.setup({}) end

  -- Register keymaps from opts.keys (skip if false)
  local k = opts.keys
  if k.jump then
    vim.keymap.set('n', k.jump, ':ObsidianQuickSwitch<CR>', { desc = 'jot: jump to note' })
  end
  if k.search then
    vim.keymap.set('n', k.search, ':ObsidianSearch<CR>', { desc = 'jot: search' })
  end
  if k.tags then
    vim.keymap.set('n', k.tags, ':ObsidianTags<CR>', { desc = 'jot: tags' })
  end
  if k.backlinks then
    vim.keymap.set('n', k.backlinks, ':ObsidianBacklinks<CR>', { desc = 'jot: backlinks' })
  end
  if k.links then
    vim.keymap.set('n', k.links, ':ObsidianLinks<CR>', { desc = 'jot: links' })
  end
  if k.new then
    vim.keymap.set('n', k.new, ':ObsidianNew<CR>', { desc = 'jot: new note' })
  end
  if k.done then
    vim.keymap.set('n', k.done, ':ObsidianToggleCheckbox<CR>', { desc = 'jot: toggle done' })
  end
  if k.follow then
    vim.keymap.set('n', k.follow, ':ObsidianFollowLink<CR>', { desc = 'jot: follow link' })
  end

  -- :JotHelp floating window
  local function show_help()
    local lines = { ' jot keybindings', '' }
    local descs = {
      jump      = 'jump to note',
      search    = 'search',
      tags      = 'tags',
      backlinks = 'backlinks',
      links     = 'links',
      new       = 'new note',
      done      = 'toggle done',
      follow    = 'follow link',
    }
    local max_lhs = 0
    local entries = {}
    for name, lhs in pairs(k) do
      if lhs then
        max_lhs = math.max(max_lhs, #lhs)
        table.insert(entries, { lhs = lhs, desc = descs[name] or name })
      end
    end
    table.sort(entries, function(a, b) return a.lhs < b.lhs end)
    for _, e in ipairs(entries) do
      table.insert(lines, '  ' .. e.lhs .. string.rep(' ', max_lhs - #e.lhs) .. '  ' .. e.desc)
    end
    table.insert(lines, '')
    table.insert(lines, ' press q to close')

    local buf = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)
    vim.bo[buf].modifiable = false
    vim.bo[buf].bufhidden = 'wipe'

    local width = 0
    for _, l in ipairs(lines) do width = math.max(width, #l) end
    width = width + 2
    local height = #lines
    local row = math.floor((vim.o.lines - height) / 2)
    local col = math.floor((vim.o.columns - width) / 2)

    vim.api.nvim_open_win(buf, true, {
      relative = 'editor',
      row = row,
      col = col,
      width = width,
      height = height,
      style = 'minimal',
      border = 'rounded',
      title = ' jot ',
      title_pos = 'center',
    })

    vim.keymap.set('n', 'q', '<cmd>close<CR>', { buffer = buf, nowait = true })
    vim.keymap.set('n', '<Esc>', '<cmd>close<CR>', { buffer = buf, nowait = true })
  end

  vim.api.nvim_create_user_command('JotHelp', show_help, { desc = 'jot: show keybindings' })
  if k.help then
    vim.keymap.set('n', k.help, show_help, { desc = 'jot: show keybindings' })
  end

  -- LuaSnip task snippet + jump keymaps
  local ls_ok, ls = pcall(require, 'luasnip')
  if ls_ok then
    local s, t, i = ls.snippet, ls.text_node, ls.insert_node
    ls.add_snippets('markdown', {
      s({ trig = '@task', wordTrig = false }, {
        t('- [ ] '),
        i(1, 'description'),
        t(' | due:'),
        i(2, 'today'),
        t(' #'),
        i(3, 'tag'),
      }),
    }, { key = 'jot' })

    local cmp_ok, cmp = pcall(require, 'cmp')
    if cmp_ok then
      local dates_src = {}
      dates_src.new = function()
        return setmetatable({}, { __index = dates_src })
      end
      dates_src.get_debug_name = function() return 'jot_dates' end
      dates_src.is_available = function() return true end
      dates_src.get_keyword_length = function() return 1 end
      dates_src.complete = function(_, _, callback)
        local items = {}
        local now = os.time()
        for d = 0, 6 do
          table.insert(items, { label = os.date('%Y-%m-%d', now + d * 86400) })
        end
        for _, w in ipairs({
          'today', 'tomorrow',
          'monday', 'mon', 'tuesday', 'tue', 'wednesday', 'wed',
          'thursday', 'thu', 'friday', 'fri', 'saturday', 'sat', 'sunday', 'sun',
          'next week',
        }) do
          table.insert(items, { label = w })
        end
        callback({ items = items, isIncomplete = false })
      end

      cmp.register_source('jot_dates', dates_src.new())

      cmp.setup.filetype('markdown', {
        sources = cmp.config.sources({
          { name = 'obsidian' },
          { name = 'obsidian_new' },
          { name = 'obsidian_tags' },
          { name = 'luasnip', keyword_pattern = [[\%(@\)\?\k\+]] },
          { name = 'jot_dates' },
          { name = 'buffer' },
          { name = 'path' },
        }),
      })
    end

    vim.keymap.set({ 'i', 's' }, '<Tab>', function()
      if ls.expand_or_jumpable() then
        ls.expand_or_jump()
      else
        vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes('<Tab>', true, true, true), 'n', false)
      end
    end, { desc = 'jot: snippet jump forward' })

    vim.keymap.set({ 'i', 's' }, '<S-Tab>', function()
      if ls.jumpable(-1) then
        ls.jump(-1)
      else
        vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes('<S-Tab>', true, true, true), 'n', false)
      end
    end, { desc = 'jot: snippet jump back' })

  end
end

return M
