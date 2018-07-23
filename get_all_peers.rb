#!/usr/bin/env ruby
require 'json'

def get_peers(peer)
  JSON.parse(`./bin/peers #{peer}`)
rescue => e
  []
end

$checked_peers = {}
$available_peers = {}
$peers = []

def check_peers(next_peer)
  while next_peer
    if !$checked_peers[next_peer]
      $checked_peers[next_peer] = true
      more_peers = get_peers(next_peer)
      if !more_peers.empty?
        $available_peers[next_peer] = true
        $peers += (more_peers - $checked_peers.keys)
        $peers.uniq!
        puts "Connected to: #{next_peer}, #{$peers.count} peers left"
      end
    end
    next_peer = $peers.pop
  end
end

initial_peer = "stellar0.keybase.io:11625"
$peers = get_peers(initial_peer)

puts "Starting 100 threads:\n"

100.times.map do |_|
  Thread.new do
    check_peers($peers.pop)
    puts "Thread done."
  end
end.each(&:join)

puts "\nDone."
puts "Got #{$checked_peers.count} peers. Of which #{$available_peers.count} were available:\n#{$available_peers.keys.join("\n")}"