use regex::Regex;
use rusqlite::functions::Context;
use rusqlite::{Connection, Error, Result};

pub fn add_regexp_function(db: &Connection) -> Result<()> {
    db.create_scalar_function(
        "regexp",
        2,
        rusqlite::functions::FunctionFlags::SQLITE_UTF8
            | rusqlite::functions::FunctionFlags::SQLITE_DETERMINISTIC,
        move |ctx: &Context| {
            let regexp: String = ctx.get(0)?;
            let text: Option<String> = match ctx.get_raw(1) {
                rusqlite::types::ValueRef::Null => None,
                rusqlite::types::ValueRef::Integer(i) => Some(i.to_string()),
                rusqlite::types::ValueRef::Real(r) => Some(r.to_string()),
                rusqlite::types::ValueRef::Text(items) => {
                    Some(String::from_utf8_lossy(items).to_string())
                }
                rusqlite::types::ValueRef::Blob(_) => None,
            };
            match text {
                Some(text) => {
                    let re = Regex::new(&regexp)
                        .map_err(|err| Error::UserFunctionError(Box::new(err)))?;

                    Ok(re.is_match(&text))
                }
                None => Ok(false),
            }
        },
    )
}
