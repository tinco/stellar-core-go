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

puts JSON.dump(nodes)
