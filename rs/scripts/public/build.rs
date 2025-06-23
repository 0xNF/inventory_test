use std::env;
use std::fs;
use std::io::{self, Write};
use std::path::{Path, PathBuf};
use std::process::{exit, Command};

// ANSI color codes
const COLOR_RESET: &str = "\x1b[0m";
const COLOR_RED: &str = "\x1b[31m";
const COLOR_GREEN: &str = "\x1b[32m";
const COLOR_YELLOW: &str = "\x1b[33m";

fn main() {
    // Get script directory (equivalent to $PSScriptRoot)
    let script_dir = env::current_exe()
        .ok()
        .and_then(|path| path.parent().map(|p| p.to_path_buf()))
        .unwrap_or_else(|| {
            print_error("Failed to determine script directory");
            exit(1);
        });

    // Default values
    let default_main_path = script_dir
        .join("..")
        .join("cmd")
        .join("mcpserver")
        .join("main.rs")
        .to_string_lossy()
        .to_string();

    let os = env::consts::OS;
    let arch = env::consts::ARCH;
    let default_output_dir = script_dir
        .join("..")
        .join("dist")
        .join(format!("{}_{}", os, arch))
        .to_string_lossy()
        .to_string();

    // Parse command line arguments
    let args: Vec<String> = env::args().collect();
    let mut main_path = default_main_path;
    let mut version = String::from("1.0.0.0");
    let mut output_dir = default_output_dir;

    let mut i = 1;
    while i < args.len() {
        match args[i].as_str() {
            "--main" => {
                if i + 1 < args.len() {
                    main_path = args[i + 1].clone();
                    i += 2;
                } else {
                    print_error("Error: --main requires a value");
                    exit(1);
                }
            }
            "--version" => {
                if i + 1 < args.len() {
                    version = args[i + 1].clone();
                    i += 2;
                } else {
                    print_error("Error: --version requires a value");
                    exit(1);
                }
            }
            "--output" => {
                if i + 1 < args.len() {
                    output_dir = args[i + 1].clone();
                    i += 2;
                } else {
                    print_error("Error: --output requires a value");
                    exit(1);
                }
            }
            _ => {
                i += 1;
            }
        }
    }

    // Print build parameters
    print_colored(COLOR_GREEN, "Build parameters:\n");
    print_colored(COLOR_YELLOW, &format!("  Main file: {}\n", main_path));
    print_colored(COLOR_YELLOW, &format!("  Version: {}\n", version));
    print_colored(
        COLOR_YELLOW,
        &format!("  Output directory: {}\n", output_dir),
    );
    println!();

    // Convert main_path string to PathBuf
    let main_path_buf = PathBuf::from(&main_path);

    // Check if main file exists
    if !main_path_buf.exists() {
        print_error(&format!("Error: Main file not found at {}", main_path));
        exit(1);
    }

    // Find project directory (directory containing Cargo.toml)
    let project_dir = if main_path_buf.is_file() {
        find_cargo_toml_dir(&main_path_buf).unwrap_or_else(|| {
            print_error("Error: Couldn't find Cargo.toml in parent directories");
            exit(1);
        })
    } else {
        main_path_buf.clone()
    };

    // Check if Cargo.toml exists in the project directory
    let cargo_toml_path = project_dir.join("Cargo.toml");
    if !cargo_toml_path.exists() {
        print_error(&format!(
            "Error: Cargo.toml not found at {}",
            cargo_toml_path.display()
        ));
        exit(1);
    }

    // Create output directory if it doesn't exist
    let output_path = PathBuf::from(&output_dir);
    if !output_path.exists() {
        fs::create_dir_all(&output_path).unwrap_or_else(|e| {
            print_error(&format!(
                "Error creating output directory {}: {}",
                output_path.display(),
                e
            ));
            exit(1);
        });
    }

    // Build the project
    print_colored(COLOR_GREEN, "Building project...\n");

    // Run cargo build with the RUSTFLAGS environment variable to set the version
    let status = Command::new("cargo")
        .current_dir(&project_dir)
        .env("RUSTFLAGS", format!("--cfg version=\"{}\"", version))
        .arg("build")
        .arg("--release") // Build in release mode
        .status()
        .unwrap_or_else(|e| {
            print_error(&format!("Failed to execute cargo build: {}", e));
            exit(1);
        });

    if !status.success() {
        print_error("Build failed");
        exit(1);
    }

    // Copy the built binary to the output directory
    let target_dir = project_dir.join("target").join("release");
    let binary_name = get_binary_name(&project_dir);
    let binary_path = target_dir.join(&binary_name);

    if !binary_path.exists() {
        print_error(&format!(
            "Error: Built binary not found at {}",
            binary_path.display()
        ));
        exit(1);
    }

    let output_binary_path = output_path.join(&binary_name);
    fs::copy(&binary_path, &output_binary_path).unwrap_or_else(|e| {
        print_error(&format!(
            "Error copying binary from {} to {}: {}",
            binary_path.display(),
            output_binary_path.display(),
            e
        ));
        exit(1);
    });

    print_colored(
        COLOR_GREEN,
        &format!(
            "Build successful! Binary saved to: {}\n",
            output_binary_path.display()
        ),
    );
}

// Helper function to print colored text
fn print_colored(color: &str, message: &str) {
    // Check if we're in a terminal that supports colors
    let use_colors = is_terminal();
    if use_colors {
        print!("{}{}{}", color, message, COLOR_RESET);
    } else {
        print!("{}", message);
    }
    io::stdout().flush().unwrap();
}

// Helper function to print error messages
fn print_error(message: &str) {
    // Check if we're in a terminal that supports colors
    let use_colors = is_terminal();
    if use_colors {
        eprintln!("{}{}{}", COLOR_RED, message, COLOR_RESET);
    } else {
        eprintln!("{}", message);
    }
}

// Find the directory containing Cargo.toml by walking up the directory tree
fn find_cargo_toml_dir(start_path: &Path) -> Option<PathBuf> {
    let mut current_dir = match start_path.parent() {
        Some(dir) => dir.to_path_buf(),
        None => return None,
    };

    loop {
        let cargo_toml_path = current_dir.join("Cargo.toml");
        if cargo_toml_path.exists() {
            return Some(current_dir);
        }

        if !current_dir.pop() {
            return None;
        }
    }
}

// Determine the binary name from the Cargo.toml file or use default
fn get_binary_name(project_dir: &Path) -> String {
    // In a full implementation, we'd parse Cargo.toml to get the actual binary name
    // For simplicity, we'll use a default name and check for exe extension on Windows
    let binary_name = "mcpserver";

    if cfg!(windows) {
        format!("{}.exe", binary_name)
    } else {
        binary_name.to_string()
    }
}

// Simple cross-platform terminal detection without external crates
fn is_terminal() -> bool {
    #[cfg(unix)]
    {
        use std::os::unix::io::AsRawFd;
        let stderr_fd = io::stderr().as_raw_fd();
        unsafe { libc::isatty(stderr_fd) != 0 }
    }

    #[cfg(windows)]
    {
        // On Windows, we'll just assume we're in a terminal
        // A more accurate check would require the Windows API
        true
    }

    #[cfg(not(any(unix, windows)))]
    {
        // For other platforms, conservatively return false
        false
    }
}

// For Unix systems, we need to define the libc extern
#[cfg(unix)]
mod libc {
    extern "C" {
        pub fn isatty(fd: i32) -> i32;
    }
}
