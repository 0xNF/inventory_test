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
            let text = ctx.get::<String>(1);
            match text {
                Ok(text) => {
                    let re = Regex::new(&regexp)
                        .map_err(|err| Error::UserFunctionError(Box::new(err)))?;

                    Ok(re.is_match(&text))
                }
                Err(e) => Ok(false),
            }
        },
    )
}
