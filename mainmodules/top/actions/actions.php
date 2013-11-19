<?php
require_once APP_ROOT.'/model/Application.php';

class topActions extends MainActions
{
	public function executeIndex()
	{
		$apps = ApplicationDb::selectAllByUpdateOrder();
		$params = array(
			'applications' => $apps,
			);
		return $this->build($params);
	}
}