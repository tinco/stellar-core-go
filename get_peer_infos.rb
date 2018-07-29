#!/usr/bin/env ruby

require 'json'

peers = JSON.load(STDIN.read)

available_peers = peers["available_peers"]
STDERR.puts "Checking #{available_peers.count} peers"

$peer_info = {}

def get_peer_info(peer_address)
  results = `./bin/peer_info #{peer_address}`
  info, connection_status = results.split(/\n/, 2)
  info = JSON.load(info)
  connection_status = JSON.load(connection_status)
  if connection_status.nil?
    if !info["ok"].nil?
      connection_status = {}
      connection_status["error"] = "Timed out"
      info = {"info" => {}}
    else
      STDERR.puts "No valid output from #{peer_address}: #{results}"
      return
    end
  end
  peers = info["peers"]
  info = info["info"]
  info["peers"] = peers
  info["accepts_connections"] = connection_status["error"].nil?
  info["error"] = connection_status["error"]
  info["address"] = peer_address
  $peer_info[info["peer_id"]] = info
end

10.times.map do |_|
  Thread.new do
    while peer = available_peers.pop
      get_peer_info(peer)
    end
  end
end.each(&:join)

puts JSON.dump($peer_info)
