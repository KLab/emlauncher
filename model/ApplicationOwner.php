<?php

/**
 * Row object for 'application_owner' table.
 */
class ApplicationOwner extends mfwObject {
	const DB_CLASS = 'ApplicationOwnerDb';
	const SET_CLASS = 'ApplicationOwnerSet';
}

/**
 * Set of ApplicationOwner objects.
 */
class ApplicationOwnerSet extends mfwObjectSet {
	public static function hypostatize(Array $row=array())
	{
		return new ApplicationOwner($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}
}

/**
 * database accessor for 'application_owner' table.
 */
class ApplicationOwnerDb extends mfwObjectDb {
	const TABLE_NAME = 'application_owner';
	const SET_CLASS = 'ApplicationOwnerSet';
}

