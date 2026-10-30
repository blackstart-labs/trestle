use std::{env, fs, path::PathBuf};

fn main() {
    println!("cargo:rerun-if-changed=web/dist");
    let dist = PathBuf::from(env::var_os("CARGO_MANIFEST_DIR").unwrap()).join("web/dist");
    assert!(
        dist.join("index.html").is_file(),
        "Frontend build missing: run `npm ci --prefix web && npm run build --prefix web` before Cargo."
    );
    let mut paths = vec![dist.join("index.html")];
    paths.extend(
        fs::read_dir(dist.join("assets"))
            .expect("Vite assets directory missing")
            .map(|entry| entry.unwrap().path()),
    );
    paths.sort();
    let mut generated = String::from("pub const ASSETS: &[(&str, &str, &[u8])] = &[\n");
    for path in paths {
        let relative = path.strip_prefix(&dist).unwrap().to_str().unwrap();
        let route = if relative == "index.html" {
            String::from("/")
        } else {
            format!("/{relative}")
        };
        assert!(
            route
                .chars()
                .all(|c| c.is_ascii_alphanumeric() || "/-_.".contains(c)),
            "Unsupported asset filename"
        );
        let mime = match path.extension().and_then(|s| s.to_str()) {
            Some("html") => "text/html; charset=utf-8",
            Some("js") => "text/javascript; charset=utf-8",
            Some("css") => "text/css; charset=utf-8",
            Some("txt") => "text/plain; charset=utf-8",
            _ => panic!("Unsupported asset type; review before shipping"),
        };
        generated.push_str(&format!(
            "({route:?}, {mime:?}, include_bytes!({:?})),\n",
            path.to_str().unwrap()
        ));
    }
    generated.push_str("];\n");
    fs::write(
        PathBuf::from(env::var_os("OUT_DIR").unwrap()).join("assets.rs"),
        generated,
    )
    .unwrap();
}
