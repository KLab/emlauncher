<?php

class logoutActions extends MainActions
{
	public function executeIndex()
	{
		$this->login_user->logout();
		return $this->redirect('/login');
	}
}
