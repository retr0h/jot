/** @type {import('prettier').Config} */
const config = {
  trailingComma: 'none',
  singleQuote: true,
  overrides: [
    {
      files: '*.md',
      options: {
        printWidth: 80,
        proseWrap: 'always'
      }
    }
  ]
}

module.exports = config
