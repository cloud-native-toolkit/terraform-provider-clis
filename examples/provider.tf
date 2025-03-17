
provider "clis" {
  bin_dir = var.bin_dir
}

resource "local_file" "bin_dir" {
  filename = "${path.cwd}/.bin_dir"

  content = var.bin_dir
}
