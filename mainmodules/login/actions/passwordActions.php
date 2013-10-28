<?php
require_once __DIR__.'/actions.php';


class passwordActions extends loginActions
{

	public function executePassword_reminder()
	{
		return $this->build();
	}

	public function executePassword_send()
	{
		return $this->redirect('/login/password_confirm');
	}

	public function executePassword_confirm()
	{
		return $this->build();
	}

	public function executePassword_reset()
	{
		return $this->build();
	}

	public function executePassword_apply()
	{
		return $this->redirect('/login/password_commit');
	}

	public function executePassword_commit()
	{
		return $this->build();
	}

}
