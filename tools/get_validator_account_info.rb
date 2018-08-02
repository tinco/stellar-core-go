#!/usr/bin/env ruby

require 'open-uri'
require 'json'

public_key = ARGV[0]
account_url = "https://horizon.stellar.org/accounts/#{public_key}"

STDERR.puts "Getting #{account_url}"
home_domain = JSON.load(open(account_url).read)["home_domain"]

home_url = "https://#{home_domain}/.well-known/stellar.toml"

STDERR.puts "Getting #{home_url}"
toml = open(home_url).read

def simplistic_toml_parse(toml)
  values = {}
  toml.lines.each do |line|
    key, value = line.split('=', 2)
    if value
      values[key.strip] = value.strip.gsub('"',"")
    end
  end
  values
end

puts simplistic_toml_parse(toml).inspect
