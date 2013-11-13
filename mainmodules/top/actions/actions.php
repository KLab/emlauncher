<?php
require_once APP_ROOT.'/model/Application.php';

class topActions extends MainActions
{
	public function executeIndex()
	{
		$params = array(
			'applications' => ApplicationDb::selectAll(),
			);
		return $this->build($params);
	}
}