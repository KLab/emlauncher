<h2>Applications</h2>

<ul>
<?php foreach($applications as $app): ?>
  <li>
    <a href="<?=url("/app?id={$app->getId()}")?>"><?=htmlspecialchars($app->getTitle())?></a>
  </li>
<?php endforeach ?>
  <li><a href="<?=url('/app/new')?>">new</a></li>
</ul>

