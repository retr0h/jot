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
      completion = { nvim_cmp = true, min_chars = 2 },
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

  -- LuaSnip @task snippets
  local ls_ok, ls = pcall(require, 'luasnip')
  if ls_ok then
    local s, t, i = ls.snippet, ls.text_node, ls.insert_node
    ls.add_snippets('markdown', {
      s('@task', {
        t('@task('),
        i(1, 'description'),
        t(' | due:'),
        i(2, 'friday'),
        t(') #'),
        i(3, 'tag'),
      }),
      s('@taskd', {
        t('@task('),
        i(1, 'description'),
        t(') #'),
        i(2, 'tag'),
      }),
    })
  end
end

return M
