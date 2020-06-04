<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/UserPass.php';

class azureadActions extends loginActions
{
	const FAILEDMAILKEY = 'azuread_failed_mail';

	public function executeAzuread()
	{
		$callback_url = mfwRequest::makeUrl('/login/azuread_callback');

		$url_base = 'https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize';
		$query = array(
			'client_id' => $this->config['azuread_app_id'],
			'redirect_uri' => $callback_url,
			'scope' => 'https://graph.microsoft.com/User.Read',
			'response_type' => 'code',
			);
		$dialog_url = mfwHttp::composeUrl($url_base,$query);

		return $this->redirect($dialog_url);
	}

	public function executeAzuread_callback()
	{
		$code = mfwRequest::param('code');

		$token = $this->getAccessToken($code);

		$userinfo = $this->getUserInfo($token);

		$mail = isset($userinfo['userPrincipalName'])? $userinfo['userPrincipalName']: null;

		apache_log('user',$mail);

		if(!$this->checkAccount($mail)){
			mfwSession::set(self::FAILEDMAILKEY,$mail);
			return $this->redirect('/login/azuread_error');
		}

		User::login($mail);
		mfwSession::clear(self::FAILEDMAILKEY);

		return $this->redirectUrlBeforeLogin();
	}

	public function executeAzuread_error()
	{
		$param = array(
			'mail' => mfwSession::get(self::FAILEDMAILKEY),
			);
		return $this->build($param);
	}

	protected function getAccessToken($code)
	{
		if(!$code){
			return null;
		}
		$callback_url = mfwRequest::makeUrl('/login/azuread_callback');

		$url_base = 'https://login.microsoftonline.com/organizations/oauth2/v2.0/token';
		$query = array(
			'code' => $code,
			'client_id' => $this->config['azuread_app_id'],
			'client_secret' => $this->config['azuread_app_secret'],
			'redirect_uri' => $callback_url,
			'grant_type' => 'authorization_code',
			'scope' => 'https://graph.microsoft.com/User.Read',
			);
		$response = mfwHttp::post($url_base,$query,['Expect:']);
		return json_decode($response,true);
	}

	protected function getUserInfo($token)
	{
		if(!isset($token['access_token'])){
			return null;
		}
		$url = 'https://graph.microsoft.com/v1.0/me';
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

