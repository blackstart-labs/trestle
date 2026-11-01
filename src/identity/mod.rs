/// Identity domain: users, sessions, CSRF, and login limits.
///
/// This module owns all identity business rules. HTTP handlers and CLI commands
/// call into this module; they do not bypass it for direct database access.
pub mod limits;
pub mod passwords;
pub mod store;
pub mod tokens;
pub mod usernames;

pub use limits::{LimitResult, LoginLimiter};
pub use passwords::Hasher;
pub use tokens::{Token, TokenHash, hash_client_token};
pub use usernames::{Username, UsernameError};
