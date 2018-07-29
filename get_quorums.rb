#!/usr/bin/env ruby

require 'json'

peers = JSON.load(STDIN.read)

threads = []
quorumsets = {}

def get_quorumsets(address)
  results = `./bin/quorumsets #{address}`
  results.split("\n").map {|qs| JSON.load(qs)}.compact
end

peers.values.each do |info|
  next unless info["accepts_connections"]
  STDERR.puts "Getting qs for: #{info["address"]}"
  threads << Thread.new do
    quorumsets[info["peer_id"]] = get_quorumsets(info["address"])
  end
end

threads.map(&:join)

puts JSON.dump(quorumsets)
