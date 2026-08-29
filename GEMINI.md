## 
### Strict Code Modification Rules
1. **Explain Before Changing**: Always explain the root cause and proposed changes in plain language before touching any files, not only in plan mode.
2. **Never Regenerate Files from Memory**: Before calling any edit tool (`client_edit_file`), you MUST view the exact current state of the file from disk using `client_view_file`.
3. **Strictly Minimal Edits**: Modify ONLY the exact lines requiring changes. Never reorder fields, alter untouched methods, reformat comments, or change unrelated imports.
4. 
