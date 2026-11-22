//! Starts the Lunar language server for Zed: `lunar lsp`, with `lunar`
//! found on the worktree's PATH.
use zed_extension_api::{self as zed, LanguageServerId, Result};

struct LunarExtension;

impl zed::Extension for LunarExtension {
    fn new() -> Self {
        LunarExtension
    }

    fn language_server_command(
        &mut self,
        _language_server_id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<zed::Command> {
        let lunar = worktree.which("lunar").ok_or_else(|| {
            "lunar is not on PATH: build it with `make install` in the Lunar repository".to_string()
        })?;
        Ok(zed::Command {
            command: lunar,
            args: vec!["lsp".to_string()],
            env: Default::default(),
        })
    }
}

zed::register_extension!(LunarExtension);
