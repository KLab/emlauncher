<?php
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Comment.php';

class topActions extends MainActions
{
	const LINE_IN_PAGE = 20;

	public function executeIndex()
	{
		$current_page = mfwRequest::param('page', 1);
		$app_count = ApplicationDb::selectCount();
		$max_page = ceil($app_count/self::LINE_IN_PAGE);
		$offset = (max(0,min($current_page,$max_page)-1)) * self::LINE_IN_PAGE;

		$apps = ApplicationDb::selectByUpdateOrderWithLimit($offset, self::LINE_IN_PAGE);

		$comments = CommentDb::selectCountsByAppIds($apps->getColumnArray('id'));

		$params = array(
			'applications' => $apps,
			'comments' => $comments,
			'cur_page' => $current_page,
			'max_page' => $max_page,
		);
		return $this->build($params);
	}
}
