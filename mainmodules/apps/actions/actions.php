<?php

class appsActions extends MainActions
{
	public function executeIndex()
	{
		//仮
		$url = mfwRequest::makeUrl('/apps/new');
		return array(array(),"<a href=\"$url\">new</a>");
	}

	public function executeNew()
	{
		$params = array(
			);
		return $this->build($params);
	}

	public function executeCreate()
	{
		$data = mfwRequest::param('icon-data');
		preg_match('/^data:([^;]*);base64,(.*)$/',$data,$match);
		$mime = $match[1];
		$base64 = $match[2];
		$png = base64_decode($base64);
		header("Content-type: $mime");
		echo $png;
	}

}