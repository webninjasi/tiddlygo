updateWikiList();
updateTemplateList();

$('form[data-live]').on('submit', function(event) {
	var $form = $(this);
	var $target = $($form.data('target'));
	var action = $form.attr('action');
	var method = $form.attr('method');
	var data = $form.serialize();

	$.ajax({
		url : action,
		method : method,
		data : data,
	}).done(function(data, textStatus, jqXHR) {
		$target.html(tplSuccess({
			data : data
		}));
		updateWikiList();
	}).fail(function(jqXHR, textStatus, errorThrown) {
		$target.html(tplError({
			data : jqXHR.responseText
		}));
	});

	event.preventDefault();
});

function updateWikiList() {
	$.getJSON("/wikilist", function(data) {
		$(".page-list").html(tplPageList(data));
	});
}

function updateTemplateList() {
	$.getJSON("/wikitemplates", function(data) {
		$("#wikitemplate").html(tplWikiTemplates(data));
	});
}
