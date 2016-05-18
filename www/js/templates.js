var tplSuccess = doT
		.template('<div class="alert alert-success"><strong>Success</strong> <span>{{=it.data}}</span></div>');
var tplError = doT
		.template('<div class="alert alert-danger"><strong>Error</strong> <span>{{=it.data}}</span></div>');
var tplPageList = doT
		.template('{{~it.pages :page:pidx}}<a href="{{=page.url}}" class="list-group-item">{{=page.name}}</a>{{~}}');
var tplWikiTemplates = doT
		.template('{{~it :tpl:idx}}<option value="{{=tpl.id}}"{{? tpl.selected }} selected{{?}}>{{=tpl.name}}</option>{{~}}');
