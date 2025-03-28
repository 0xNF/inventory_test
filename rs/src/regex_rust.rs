use rusqlite::functions::Context;
use rusqlite::{Connection, Error, Result};
use regex::Regex;

pub fn add_regexp_function(db: &Connection) -> Result<()> {
    db.create_scalar_function(
        "regexp",
        2,
        rusqlite::functions::FunctionFlags::SQLITE_UTF8 | rusqlite::functions::FunctionFlags::SQLITE_DETERMINISTIC,
        move |ctx: &Context| {
            let regexp: String = ctx.get(0)?;
            let text: String = ctx.get(1)?;
            
            let re = Regex::new(&regexp)
                .map_err(|err| Error::UserFunctionError(Box::new(err)))?;

            Ok(re.is_match(&text))
        },
    )
}