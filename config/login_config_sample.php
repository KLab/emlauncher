/**@file
 * ログイン設定.
 * login_config.phpにリネームして使ってください.
 */
$login_config = array(
	'local' => array(
		/**
		 * email+passwordによるログインを許可.
		 * `user_pass`テーブルに登録されているアカウントでログイン可能にする.
		 * @note
		 *  ユーザを追加する時は`user_pass`に`email`のみを登録し
		 *  パスワードリセットの手順を踏むことでパスワードを登録する.
		 */
		'enable_password' => true,
		/**
		 * パスワードリマインダのメール送信用アドレス.
		 */
		'reminder_address' => 'no-reply@example.com',

		/**
		 * Googleアカウントでのログインを許可.
		 * アカウントのメールアドレスが'allowed_mailaddr_pattern'にマッチするか,
		 * user_passテーブルに存在したらログインを認める.
		 */
		'enaable_google_auth' => true,
		'google_app_id' => 'xxxxxxxx.apps.googleusercontent.com',
		'google_app_secret' => 'xxxxxxxx',
		/**
		 * Googleアカウントでのログインを許可するメールアドレスパターン.
		 */
		'allowed_mailaddr_pattern' => '/@klab\.com$/',
		),
	);

// awsもlocalと同じ設定を使う
$login_config['aws'] = $login_config['local'];

