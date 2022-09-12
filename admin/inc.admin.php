<?php

class admin {
	static public $host = SOLPROXY_HOST;
	static public $last_error = false;
	
	static private function run($url)
	{
		self::$last_error = false;
		
		$data = file_get_contents($url);
		$data = json_decode($data, true);
		if (!is_array($data)) {
			$data = ['error'=>'Unknown error'];
		}
		if (isset($data['error']))
			self::$last_error = $data['error'];
		return $data;
	}
	
	static function nodeList() {
		self::$last_error = false;
		return self::run(self::$host."?action=solana_admin");
	}
	
	static function nodeAdd($config_json, $replace_node_id = false)
	{
		self::$last_error = false;
		$config_json = urlencode($config_json);
		return self::run(self::$host."?action=solana_admin_add&node={$config_json}&remove_id={$replace_node_id}");
	}
	
	static function nodeRemove($node_id = false)
	{
		self::$last_error = false;
		$data = self::run(self::$host."?action=solana_admin_remove&id={$node_id}");
		return $data;
	}
}

/*
echo "<pre>";
print_r(admin::nodeList());
echo "<hr>";
print_r(admin::nodeAdd('{"url":"https://rpc.ankr.com/solana", "public":true, "throttle":"r,30,90;f,5,150;d,200000,5;r,3600,1000", "score_modifier":-19999}', 1));
echo "<hr>";
print_r(admin::nodeList());
echo "<hr>";
print_r(admin::nodeRemove(3));
*/