<?php

/**
 * Paging parameters
 */
class Paging {

	protected $current_page_number;
	protected $total_page_number;

	function __construct($current_page_number = 1, $total_page_number = 1) {
		$this->total_page_number = max(1, $total_page_number);
		$this->current_page_number = max(1, min($current_page_number, $this->total_page_number));
	}

	public function getCurrentPage() {
		return $this->current_page_number;
	}
	public function getTotalPageNumber() {
		return $this->total_page_number;
	}
}

class InfinitePaging {
	protected $current_page_number;
	protected $has_next_page;

	function __construct($current_page_number = 1, $has_next_page = false) {
		$this->current_page_number = $current_page_number;
		$this->has_next_page = $has_next_page;
	}

	public function getCurrentPage() {
		return $this->current_page_number;
	}
	public function hasNextPage() {
		return $this->has_next_page;
	}
}
