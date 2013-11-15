<?php
require_once __DIR__.'/actions.php';

class preferenceActions extends appActions
{
	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}
		if(!$this->app->isOwner($this->login_user)){
			return $this->buildErrorPage(
				'Permission Denied',array(self::HTTP_403_FORBIDDEN));
		}
		return null;
	}

	public function executePreference()
	{
		$params = array(
			);
		return $this->build($params);
	}


}
