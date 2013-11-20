New package "<?=$pkg->getTitle()?>" was uploaded to "<?=$app->getTitle()?>".

<?php if($pkg->getDescription()): ?>
"<?=$pkg->getDescription()?>"
<?php endif ?>

Platform: <?=$pkg->getPlatform()?>

Tags:<?php foreach($pkg->getTags() as $tag): ?>
  <?=$tag->getName()?>
<?php endforeach ?>


Package URL: <?=$package_url?>

