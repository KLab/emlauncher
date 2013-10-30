<?php

class logoutActions extends MainActions
{
	public function executeIndex()
	{
		$this->login_user->logout();
		$this->clearUrlBeforelogin();
		return $this->redirect('/login');
	}
}
