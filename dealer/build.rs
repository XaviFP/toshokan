fn main() -> Result<(), Box<dyn std::error::Error>> {
        tonic_build::configure()
             .build_server(true)
             .out_dir("api/proto/v1")
             .compile(
                 &["api/proto/v1/dealer.proto"],
                 &["api/proto/v1/"],
             )?;
        Ok(())
}