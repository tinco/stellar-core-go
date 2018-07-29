#!/usr/bin/env ruby

peers = JSON.load($STDIN.read)

available_peers = peers["available_peers"]

$peer_info = {}

def get_peer_info(peer_address)
  results = `./bin/peer_info #{peer_address}`
  info, connection_status = results.split(/\n/, 2)
  info = JSON.load(info)
  connection_status = JSON.load(connection_status)
  info["accepts_connections"] = connection_status["error"].nil?
  info["error"] = connection_status["error"]
  info["address"] = peer_address
  $peer_info[info["peer_id"]] = info
end

10.times.map do |_|
  Thread.new do
    peer = available_peers.pop
    get_peer_info(peer)
  end
end.each(&:join)

puts JSON.dump($peer_info)
