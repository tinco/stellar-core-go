#!/usr/bin/env ruby

require 'open-uri'
require 'json'

url = "https://raw.githubusercontent.com/stellar/dashboard/master/common/nodes.js"
response = open(url).read

# WARNING: this is almost as horrible as running javascript ;)
nodes = proc do
  $SAFE = 1
  eval response.gsub("module.exports =","")
end.call

known_validators = {}

nodes.each do |n|
  known_validators[n[:publicKey]] = n
  n.delete(:publicKey)
end

raise "Double Public Key entry" unless nodes.count == known_validators.count

puts JSON.dump(known_validators)
