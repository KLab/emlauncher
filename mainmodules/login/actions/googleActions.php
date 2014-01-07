<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/UserPass.php';

class googleActions extends loginActions
{

	public function executeGoogle()
	{
		$callback_url = mfwRequest::makeUrl('/login/google_callback');

		$url_base = 'https://accounts.google.com/o/oauth2/auth';
		$query = array(
			'client_id' => $this->config['google_app_id'],
			'redirect_uri' => $callback_url,
			'scope' => 'https://www.googleapis.com/auth/userinfo.email',
			'response_type' => 'code',
			'approval_prompt' => 'auto',
			);
		$dialog_url = mfwHttp::composeUrl($url_base,$query);

		return $this->redirect($dialog_url);
	}

	public function executeGoogle_callback()
	{
		$code = mfwRequest::param('code');

		$token = $this->getAccessToken($code);

		$userinfo = $this->getUserInfo($token);

		$mail = isset($userinfo['email'])? $userinfo['email']: null;

		if(!$this->checkAccount($mail)){
			return $this->redirect('/login/google_error');
		}

		User::login($mail);
		apache_log('user',$mail);

		return $this->redirectUrlBeforeLogin();
	}

	public function executeGoogle_error()
	{
		return $this->build();
	}

	protected function getAccessToken($code)
	{
		if(!$code){
			return null;
		}
		$callback_url = mfwRequest::makeUrl('/login/google_callback');

		$url_base = 'https://accounts.google.com/o/oauth2/token';
		$query = array(
			'code' => $code,
			'client_id' => $this->config['google_app_id'],
			'client_secret' => $this->config['google_app_secret'],
			'redirect_uri' => $callback_url,
			'grant_type' => 'authorization_code',
			);
		$response = mfwHttp::post($url_base,$query);
		return json_decode($response,true);
	}

	protected function getUserInfo($token)
	{
		if(!isset($token['access_token'])){
			return null;
		}
		$url = 'https://www.googleapis.com/oauth2/v1/userinfo';
		$header = array(
			"Authorization: {$token['token_type']} {$token['access_token']}",
			);
		$response = mfwHttp::get($url,$header);
		return json_decode($response,true);
	}

	protected function checkAccount($mail)
	{
		if(isset($this->config['allowed_mailaddr_pattern'])){
			// パターンマッチしたらOK
			if(preg_match($this->config['allowed_mailaddr_pattern'],$mail)){
				return true;
			}
		}
		// user_passテーブルに登録されていたらOK
		$user = UserPassDb::selectByEmail($mail);

		return ($user!=null);
	}

}

