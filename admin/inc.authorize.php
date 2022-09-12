<?php

class log
{
	const FLUSH_ALWAYS = 'always';
	
	static public function getLogPath($filename, $dir = false)
	{
		if ($dir === false)
			$dir = getcwd();
		
		// use current directory
		$path = "{$dir}/{$filename}.log.txt";
		
		// if we're not on dev server - let's place logs in safe path
		if (DEV_SERVER)
			return $path;
		if (strpos($path, '/tools/') !== false || strpos($path, '/cron/') !== false || strpos($path, '/logs/') !== false)
			return $path;
		
		$path = explode("/htdocs/", "{$dir}/");
		$path = implode("/logs/system/", $path);
		$path = str_replace("//", "/", "{$path}/{$filename}.log.txt");
		
		$dir = dirname($path);
		if (!is_dir($dir))
			@mkdir($dir, 0777, true);
		return $path;
	}
	
	private static $lg = false;
	private $buffer = array();
	private $buffer_html = array();
	
	private $last_logged = false;
	private $update_interval = false;
	
	private $ts = false;
	private $ts_group = array();
	
	private $ts_start = false;
	
	static function GetLog($filename, $standard_headers = false, $update_interval = false)
	{
		static $cache = [];
		if (isset($cache[$filename]))
			return $cache[$filename];
		
		$n = new log($filename, $standard_headers, $update_interval);
		$cache[$filename] = $n;
		return $n;
	}
	
	/**
	 * Construct a log object. If file name isn't specified, it'll result in building a memory log
	 * @param mixed $filename Name of the file, false for memory log
	 * @param bool $standard_headers Inculde standard header, containing time at the begin of new log
	 * @param mixed $update_interval Time after which current data from the log should be saved to file and flushed
	 */
	function __construct($filename = false, $standard_headers = false, $update_interval = false)
	{
		if ($filename === false && $update_interval !== false)
		{
			error_log("Warning: Memory log needs to have update_interval set to FALSE!");
			$update_interval = false;
		}
		
		if ($filename === false)
			$this->file = false;
		else
			$this->file = self::getLogPath($filename);
		
		$this->last_logged = time();
		$this->update_interval = $update_interval;
		$this->ts = new TimeSpan();
		$this->ts_group = array();
		
		if ($standard_headers)
		{
			$this->ts_start = new TimeSpan();
			$this->log("");
			$this->log("New task started: ".date("Y-m-d H:i:s"));
		}
	}
	
	function flush($write_output = true)
	{
		$force_flush = $write_output === self::FLUSH_ALWAYS;
		if ( ($write_output || $force_flush) && $this->file !== false)
		{
			if (defined("DISABLE_LOGGING") && DISABLE_LOGGING && !$force_flush)
				return false;
			
			$f = @fopen($this->file, 'a');
			if ($f === false)
				return false;
			flock($f, LOCK_EX);
			fwrite($f, implode("", $this->buffer));
			flock($f, LOCK_UN);
			fclose($f);
		}
		
		$this->last_logged = time();
		$ret = array("txt"=>$this->buffer, "html"=>$this->buffer_html);
		
		// clean the buffers!
		$this->buffer = array();
		
		return $ret;
	}
	
	function __destruct()
	{
		if ($this->ts_start !== false)
		{
			$took = $this->ts_start->getTimeSpanF();
			$this->log("============== Took: {$took} ==============");
		}
		$this->flush();
	}
	
	private function _log($text, $newline, $class = false)
	{
		if (is_array($text))
		{
			if ($newline)
			{
				foreach ($text as $v)
					$this->_log($v, $newline, $class);
				return;
			}
			else
				$text = implode(",", $text);
		}
		
		$text = str_repeat(" ", count($this->ts_group)*4).$text;
		$this->buffer[] = ($text.($newline ? "\n" : ""));
		
		$html = $text;
		if ($class != '' && $class !== false)
			$html = "<span class='{$class}'>$html</span>";
		if ($newline)
			$html .= "\n";
		$this->buffer_html[] = $html;
		
		if ($this->update_interval !== false && (time() - $this->last_logged) >= $this->update_interval)
			$this->flush();
	}
	
	function log($text, $newline = true)
	{
		$this->_log($text, $newline);
	}
	
	function warn($text, $newline = true)
	{
		$this->_log("warning: {$text}", $newline, 'warning');
	}
	
	function err($text, $newline = true)
	{
		$this->_log("error: {$text}", $newline, 'error');
	}
	
	function logImport(log $log)
	{
		if (get_class($log) != get_class($this))
		{
			error_log("Can't import log, incorrect variable type");
			return false;
		}
		
		$pad = str_repeat(" ", count($this->ts_group)*4);
		$content = $log->flush(false);
		foreach ($content['txt'] as $v)
			$this->buffer[] = $pad.$v;
		
		foreach ($content['html'] as $v)
			$this->buffer_html[] = $pad.$v;
		
		return count($content['txt']);
	}
	
	function logTime($text = false, $newline = true)
	{
		$took = $this->ts->getTimeSpanF();
		$this->ts = new TimeSpan();
		if ($text === false)
			return;
		
		$pad = str_repeat(" ", max(0, 7-strlen($took)));
		$this->_log("{$took}{$pad} {$text}", $newline);
	}
	
	function groupStart($header)
	{
		$this->ts = $this->ts_group[] = new TimeSpan();
		$this->_log("# {$header} {", true, 'header');
	}
	
	function groupEnd($footer = '', $display_time = false)
	{
		$ts = $this->ts_group[count($this->ts_group)-1];
		if ($display_time)
			$footer = "# Took: ".$ts->getTimeSpanF()." / {$footer}";
		else
			$footer = "# }";
		
		$this->log($footer);
		array_pop($this->ts_group);
	}
	
	function getHTML($flush_buffer = true)
	{
		$style = "<style>.error {color: red;} .header {background: #dddddd; font-weight: bolder;} .warning {color: #ff7700}</style>";
		$ret = "{$style}\n<pre>".implode("", $this->buffer_html)."</pre>";
		
		if ($flush_buffer)
			$this->buffer_html = [];
		
		return $ret;
	}
	
	function err_backtrace($error)
	{
		$stack = debug_backtrace();
		unset($stack[0]);
		$stack = array_values($stack);
		
		if (!is_array($error))
			$error = [$error];
		
		$log[] = "";
		$log[] = "------------------";
		$log[] = "Error on: ".date("Y-m-d H:i:s");
		foreach ($error as $k=>$v)
			$log[] = $v;
		
		if (isset($_SERVER['REQUEST_URI']))
			$log[] = "URL: {$_SERVER['HTTP_HOST']}{$_SERVER['REQUEST_URI']}";
		$log[] = "Stack trace:";
		
		foreach ($stack as $num=>$v)
		{
			$num++;
			
			// $v['args'] is reference, so this part which looks weird is actually very needed
			$args = [];
			foreach ($v['args'] as $_kk=>$_vv)
				$args[$_kk] = $_vv;
			
			foreach ($args as $kk=>$vv)
			{
				if (is_array($vv) || is_object($vv))
					$vv = str_replace("'", "\"", "@".json_encode($vv));
				else
				{
					if (!(is_numeric($vv) || is_float($vv)))
					{
						$vv = str_replace("'", "\"", $vv);
						$vv = "'{$vv}'";
					}
				}
				
				if (strlen($vv) > 50)
					$vv = substr($vv, 0, 47).'...';
				$args[$kk] = $vv;
			}
			$args = implode(", ", $args);
			
			if (!isset($v['file']))
				$v['file'] = '';
			if (!isset($v['line']))
				$v['line'] = '-';
			$v['file'] = str_replace($_SERVER['DOCUMENT_ROOT'], '', $v['file']);
			
			$obj = '';
			if (isset($v['object']))
				$obj = get_class($v['object']).$v['type'];
			
			$log[] = "{$num}. {$v['file']}:{$v['line']} {$obj}{$v['function']}($args)";
		}
		
		$log[] = '------------------';
		$this->_log(implode("\n", $log), true, 'error');
		$this->flush(self::FLUSH_ALWAYS);
	}
	
	
	private static $status_id = false;
	static function statusRegisterThread($id, $thread_name, $max_threads = 5)
	{
		static $registered = false;
		if ($registered !== false)
			return true;
		$registered = true;
		
		$num = 0;
		do
		{
			$num ++;
			if ($num > $max_threads)
				return false;
			
			$alloc_key = "XK~REG~Lock~{$id}:{$thread_name}:{$num}";
		} while(db::single("SELECT GET_LOCK($$, 0)", $alloc_key) != 1);
		
		self::$status_id = [$id, $thread_name, $num];
		
		self::statusSet('Started');
		register_shutdown_function(function () {
			self::statusSet(false);
		});
		
		return $num;
	}
	
	static function statusSet($status)
	{
		if (self::$status_id === false)
			return false;
		
		static $last_updated = false;
		static $started = false;
		static $uuid = false;
		if ($uuid === false)
		{
			$posix_proc_file = "/proc/".getmypid();
			if (!is_dir("/proc/") || !file_exists($posix_proc_file))
				$posix_proc_file = '-';
			$uuid = "{$posix_proc_file}|".uniqid("", true);
		}
		if ($last_updated === false || time() - $last_updated > 600)
		{
			$last_updated = time();
			if ($started === false)
				$started = time();
			
			$id = self::$status_id[0];
			simplememlist::$smc = false;	// important - reset memlist creation time, so we'll have keys re-generated
			$memlist = new simplememlist("thstatus-{$id}", 0, 3600);
			$memlist->addElementCounted(json_encode(self::$status_id));
		}
		
		$limit = $unit = strtolower(ini_get('memory_limit'));
		$unit = trim($limit, "b0123456789 \t\n\r");
		$limit = trim($limit, "kmgb");
		if (strpos($unit, 'k')!==false)
			$limit *= 1024;
		if (strpos($unit, 'm')!==false)
			$limit *= 1024*1024;
		if (strpos($unit, 'g')!==false)
			$limit *= 1024*1024*1024;
		
		$mem_used = memory_get_usage(false)." / {$limit}";
		$key = "status-".implode(":", self::$status_id);
		if ($status !== false)
			__getMemcache()->set($key, json_encode(['started'=>$started, 'comment'=>$status, 'uuid'=>$uuid, 'memory'=>$mem_used]), 0, 3600);
		else
			__getMemcache()->set($key, json_encode(['started'=>$started, 'finished'=>time(), 'comment'=>false]), 0, 3600);
	}
	
	// this function will look for the process in linux, and if it can't find it...
	// it'll report the task as broken!
	static private function hasProcessBroken($data)
	{
		if (!isset($data['uuid']))
			return false;
		
		$uuid = $data['uuid'];
		if ( ($_t = __getMemcache()->get($uuid)) > 0)
			return $_t;
		
		static $last_check = false;
		if (! ($last_check === false || time() - $last_check > 3))
			return false;
		$last_check = time();
		
		if (file_exists(explode("|", $uuid)[0]))
			return false;
		
		__getMemcache()->add($uuid, $last_check, 0, 3600*12);
		return $last_check;
	}
	
	
	static function statusGet($id)
	{
		$memlist = new simplememlist("thstatus-{$id}", 0, 3600);
		$memlist2 = new simplememlist("thstatus-{$id}", 1, 3600);
		
		$all_keys = [];
		foreach ($memlist->getList() as $v)
			$all_keys[$v['element']] = true;
		foreach ($memlist2->getList() as $v)
			$all_keys[$v['element']] = true;
		$all_keys = array_keys($all_keys);
		
		// generate... id:thread => max_num
		$tmp = [];
		foreach ($all_keys as $v)
		{
			$v = json_decode($v, true);
			$k = "{$v[0]}:{$v[1]}";
			if (!isset($tmp[$k]))
				$tmp[$k] = $v[2];
			else
				$tmp[$k] = max($tmp[$k], $v[2]);
		}
		
		uksort($tmp, function($a, $b){
			return strcasecmp($a, $b);
		});
		
		$status = [];
		foreach ($tmp as $v=>$max_num)
		{
			
			$vv = explode(":", $v);
			$_id = $vv[0];
			$_thread = implode(":", array_slice($vv, 1));
			
			for ($num=1; $num<=$max_num; $num++)
			{
				$data = __getMemcache()->get("status-{$v}:{$num}");
				if ($data !== false)
					$data = json_decode($data, true);
				if ($data === false)
					$data = ['comment'=>'Thread Finished', 'running'=>false];
				else
					$data['running'] = true;
				
				// check if the process broken unexpectedly
				if ( ($broken = self::hasProcessBroken($data)) !== false)
					$data['finished'] = $broken;
				// <<
				
				if (isset($data['finished']))
				{
					$f = time() - $data['finished'];
					$t = $data['finished'] - $data['started'];
					if ($broken === false)
						$data['comment'] = "Thread Finished {$f}s ago (Took {$t}s)";
					else
						$data['comment'] = "Error: Thread finished unexpectedly ~{$f}s ago (Took ~{$t}s)";
					$data['running'] = false;
				}
				elseif (isset($data['started']) && false)
				{
					$t = time() - $data['started'];
					$data['comment'] .= " (Took {$t}s)";
				}
				
				$tmp['id'] = $_id;
				$tmp['thread'] = $_thread;
				$tmp = array_merge($tmp, $data);
				
				$status["{$v} #{$num}"] = $data;
			}
		}
		
		return $status;
	}
	
}

class authorize
{
	private $credentials;
	private $realm;
	private $noauth_text;
	private $auth_callbacks = [];
	
	private $allow_local = false;
	
	function __construct($credentials = null, $realm = 'Restricted Area', $noauth_text = 'Authorization Required')
	{
		$this->credentials = $credentials;
		$this->realm = $realm;
		$this->noauth_text = $noauth_text;
	}
	
	private function is_local_call()
	{
		if (DEV_SERVER) {
			return true;
		}
		
		if (strtolower(php_sapi_name()) === 'cli')
			return "linux shell";
		
		if (!isset($_SERVER['SERVER_ADDR']) || !isset($_SERVER['REMOTE_ADDR'])) {
			return false;
		}
		
		$ip = $_SERVER['SERVER_ADDR'];
		$remote_ip = $_SERVER['REMOTE_ADDR'];
		
		if (!filter_var($ip, FILTER_VALIDATE_IP) || !filter_var($remote_ip)) {
			return false;
		}
		
		if (strlen($ip) < 5 || strlen($remote_ip) < 5) {
			return false;
		}
		
		$bad_vars = ['HTTP_X_FORWARDED_FOR', 'HTTP_CLIENT_IP'];
		foreach ($bad_vars as $v) {
			if (isset($_SERVER[$v])) {
				return false;
			}
		}
		
		$ip = explode(".", $ip);
		array_pop($ip);
		$ip = implode(".", $ip);
		
		$remote_ip = explode(".", $remote_ip);
		array_pop($remote_ip);
		$remote_ip = implode(".", $remote_ip);
		
		if (strcmp($remote_ip, $ip) === 0) {
			return "local call";
		}
		return false;
	}
	
	function allow_local_calls($allow = true)
	{
		$this->allow_local = $allow;
	}
	
	function auth_callback($function)
	{
		$this->auth_callbacks[] = $function;
	}
	
	private function log_auth($is_ok, $user, $error_msg = '')
	{
		if ($is_ok)
		{
			$log = new log("auth-ok", false);
			$message = "Authorized ";
		}
		else
		{
			$log = new log("auth-failed", false);
			$message = "Auth. failed ";
		}
		
		$ip = isset($_SERVER['REMOTE_ADDR']) ? $_SERVER['REMOTE_ADDR'] : "?" ;
		$uri = isset($_SERVER['SERVER_NAME']) ? $_SERVER['SERVER_NAME'] : "?:/";
		$uri .= "/".isset($_SERVER['SCRIPT_NAME']) ? $_SERVER['SCRIPT_NAME'] : "?";
		$uri .= isset($_SERVER['QUERY_STRING']) && strlen($_SERVER['QUERY_STRING']) > 0 ? "?{$_SERVER['QUERY_STRING']}" : "";
		
		if ($error_msg != '')
			$error_msg = "\t/{$error_msg}";
		
		$date = date("Y-m-d H:i:s");
		$log->log("{$message} @{$date} for IP:{$ip}, Entity:{$user}, URI:{$uri}{$error_msg}");
	}
	
	private function show_auth_popup($error_msg = false)
	{
		$auth_header = 'WWW-Authenticate: Digest realm="##1##",qop="##2##",nonce="##3##", opaque=##4##';
		$auth_header = str_replace(["##1##", "##2##", "##3##", "##4##"], [$this->realm, 'auth', uniqid(), md5($this->realm)], $auth_header);
		
		header('HTTP/1.1 401 Unauthorized');
		header($auth_header);
		
		die($this->noauth_text."<br>  <i>$error_msg</i>");
	}
	
	function authorize()
	{
		// do we allow local calls
		if ($this->allow_local) {
			$ret = $this->is_local_call();
			if ($ret !== false) {
				$this->log_auth(true, "local({$ret})");
				return true;
			}
		}
		
		// auth modules support
		foreach ($this->auth_callbacks as $f)
		{
			$ret = $f();
			if ($ret !== false)
			{
				$this->log_auth(true, "callback({$ret})");
				return true;
			}
		}
		
		if (!isset($_SERVER['PHP_AUTH_DIGEST']) || strlen($_SERVER['PHP_AUTH_DIGEST']) == 0) {
			$ret = $this->is_local_call();
			$ret = $ret !== false ? $ret : "outside";
			$this->log_auth(true, "empty_auth({$ret})");
			
			$this->show_auth_popup();
		}
		
		// analyze the PHP_AUTH_DIGEST variable
		$data = self::http_digest_parse($_SERVER['PHP_AUTH_DIGEST']);
		$login_user = $data['username'];
		
		if (!isset($this->credentials[$login_user]))
		{
			$err_msg = "Wrong Credentials 0x0001";
			$this->log_auth(false, "not found ({$login_user})", $err_msg);
			$this->show_auth_popup($err_msg);
		}
		
		// generate the valid response
		$pass = $this->credentials[$login_user];
		$A1 = md5("{$login_user}:{$this->realm}:{$pass}");
		$A2 = md5("{$_SERVER['REQUEST_METHOD']}:{$data['uri']}");
		$valid_response = md5("{$A1}:{$data['nonce']}:{$data['nc']}:{$data['cnonce']}:{$data['qop']}:{$A2}");
		
		if ($data['response'] != $valid_response)
		{
			$err_msg = "Wrong Credentials 0x0002";
			$this->log_auth(false, $login_user, $err_msg);
			$this->show_auth_popup($err_msg);
		}
		
		$this->log_auth(true, $login_user);
		
		return true;
	}
	
	
	static private function http_digest_parse($txt)
	{
		// protect against missing data
		$needed_parts = array('nonce'=>1, 'nc'=>1, 'cnonce'=>1, 'qop'=>1, 'username'=>1, 'uri'=>1, 'response'=>1);
		$data = array();
		$keys = implode('|', array_keys($needed_parts));
		
		preg_match_all('@(' . $keys . ')=(?:([\'"])([^\2]+?)\2|([^\s,]+))@', $txt, $matches, PREG_SET_ORDER);
		foreach ($matches as $m) {
			$data[$m[1]] = $m[3] ? $m[3] : $m[4];
			unset($needed_parts[$m[1]]);
		}
		
		return $needed_parts ? false : $data;
	}
}

class TimeSpan
{
	var $start_time;
	
	function __construct($start_now = true)
	{
		$this->start_time = 0;
		if ($start_now) $this->start();
	}
	
	function start()
	{
		$this->start_time = $this->getmicrotime();
	}
	
	function getTimeSpanMS($format_for_output = true)
	{
		$ret = ($this->getmicrotime() - $this->start_time)*1000;
		if ($format_for_output)
			$ret = sprintf("%0.2f", $ret);
		return $ret;
	}
	
	function getTimeSpanF()
	{
		$took = $this->getTimeSpanMS(false);
		if ($took >= 1000)
			$took = sprintf("%.1f", ((float)$took) / 1000.0)."s";
		else
			$took = sprintf("%.2fms", $took);
		
		return $took;
	}
	
	function getmicrotime()
	{
		list($usec, $sec) = explode(" ", microtime());
		return ((float)$usec + (float)$sec);
	}
}