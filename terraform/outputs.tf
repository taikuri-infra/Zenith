output "freezenith_records" {
  description = "freezenith.com DNS records"
  value = {
    root       = cloudflare_record.freezenith_root.hostname
    www        = cloudflare_record.freezenith_www.hostname
    api        = cloudflare_record.freezenith_api.hostname
    demo_ms    = cloudflare_record.freezenith_demo_ms.hostname
    demo_cloud = cloudflare_record.freezenith_demo_cloud.hostname
  }
}

output "embermind_records" {
  description = "embermind.app DNS records"
  value = {
    ms    = cloudflare_record.embermind_ms.hostname
    cloud = cloudflare_record.embermind_cloud.hostname
  }
}
