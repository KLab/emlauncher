<?php
require_once APP_ROOT.'/model/Application.php';

class topActions extends MainActions
{
	const LINE_IN_PAGE = 20;

	public function executeIndex()
	{
		$current_page = mfwRequest::param('page', 1);
		$app_count = ApplicationDb::selectCount();
		$paging = $this->createPaging($current_page, $app_count, self::LINE_IN_PAGE);

		$offset = ($paging->getCurrentPage() - 1) * self::LINE_IN_PAGE;
		$apps = ApplicationDb::selectByUpdateOrderWithLimit($offset, self::LINE_IN_PAGE);

		$params = array(
			'applications' => $apps,
			'paging' => $paging,
		);
		return $this->build($params);
	}
}
