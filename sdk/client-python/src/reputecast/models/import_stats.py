from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.import_list_after import ImportListAfter
  from ..models.import_stats_reasons import ImportStatsReasons
  from ..models.import_stats_suppressed_by import ImportStatsSuppressedBy
  from ..models.import_stats_texts import ImportStatsTexts
  from ..models.import_template_stats import ImportTemplateStats





T = TypeVar("T", bound="ImportStats")



@_attrs_define
class ImportStats:
    """ 
        Attributes:
            confirms_sent (int): Re-permission emails sent through the System stream
            duration_ms (int):
            imported (int):
            lists (list[ImportListAfter]):
            merged (int):
            pending (int):
            pushed_to (list[str]): Providers the new suppressions were pushed to
            reasons (ImportStatsReasons):
            rows (int):
            skipped (int):
            suppressed (int):
            suppressed_by (ImportStatsSuppressedBy): unsubscribe, hard_bounce, complaint
            templates (ImportTemplateStats | Unset):
            texts (ImportStatsTexts | Unset):
     """

    confirms_sent: int
    duration_ms: int
    imported: int
    lists: list[ImportListAfter]
    merged: int
    pending: int
    pushed_to: list[str]
    reasons: ImportStatsReasons
    rows: int
    skipped: int
    suppressed: int
    suppressed_by: ImportStatsSuppressedBy
    templates: ImportTemplateStats | Unset = UNSET
    texts: ImportStatsTexts | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_list_after import ImportListAfter # noqa: PLC0415
        from ..models.import_stats_reasons import ImportStatsReasons # noqa: PLC0415
        from ..models.import_stats_suppressed_by import ImportStatsSuppressedBy # noqa: PLC0415
        from ..models.import_stats_texts import ImportStatsTexts # noqa: PLC0415
        from ..models.import_template_stats import ImportTemplateStats # noqa: PLC0415
        confirms_sent = self.confirms_sent

        duration_ms = self.duration_ms

        imported = self.imported

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        merged = self.merged

        pending = self.pending

        pushed_to = self.pushed_to



        reasons = self.reasons.to_dict()

        rows = self.rows

        skipped = self.skipped

        suppressed = self.suppressed

        suppressed_by = self.suppressed_by.to_dict()

        templates: dict[str, Any] | Unset = UNSET
        if not isinstance(self.templates, Unset):
            templates = self.templates.to_dict()

        texts: dict[str, Any] | Unset = UNSET
        if not isinstance(self.texts, Unset):
            texts = self.texts.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "confirmsSent": confirms_sent,
            "durationMs": duration_ms,
            "imported": imported,
            "lists": lists,
            "merged": merged,
            "pending": pending,
            "pushedTo": pushed_to,
            "reasons": reasons,
            "rows": rows,
            "skipped": skipped,
            "suppressed": suppressed,
            "suppressedBy": suppressed_by,
        })
        if templates is not UNSET:
            field_dict["templates"] = templates
        if texts is not UNSET:
            field_dict["texts"] = texts

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_list_after import ImportListAfter # noqa: PLC0415
        from ..models.import_stats_reasons import ImportStatsReasons # noqa: PLC0415
        from ..models.import_stats_suppressed_by import ImportStatsSuppressedBy # noqa: PLC0415
        from ..models.import_stats_texts import ImportStatsTexts # noqa: PLC0415
        from ..models.import_template_stats import ImportTemplateStats # noqa: PLC0415
        d = dict(src_dict)
        confirms_sent = d.pop("confirmsSent")

        duration_ms = d.pop("durationMs")

        imported = d.pop("imported")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = ImportListAfter.from_dict(lists_item_data)



            lists.append(lists_item)


        merged = d.pop("merged")

        pending = d.pop("pending")

        pushed_to = cast(list[str], d.pop("pushedTo"))


        reasons = ImportStatsReasons.from_dict(d.pop("reasons"))




        rows = d.pop("rows")

        skipped = d.pop("skipped")

        suppressed = d.pop("suppressed")

        suppressed_by = ImportStatsSuppressedBy.from_dict(d.pop("suppressedBy"))




        _templates = d.pop("templates", UNSET)
        templates: ImportTemplateStats | Unset
        if isinstance(_templates,  Unset):
            templates = UNSET
        else:
            templates = ImportTemplateStats.from_dict(_templates)




        _texts = d.pop("texts", UNSET)
        texts: ImportStatsTexts | Unset
        if isinstance(_texts,  Unset):
            texts = UNSET
        else:
            texts = ImportStatsTexts.from_dict(_texts)




        import_stats = cls(
            confirms_sent=confirms_sent,
            duration_ms=duration_ms,
            imported=imported,
            lists=lists,
            merged=merged,
            pending=pending,
            pushed_to=pushed_to,
            reasons=reasons,
            rows=rows,
            skipped=skipped,
            suppressed=suppressed,
            suppressed_by=suppressed_by,
            templates=templates,
            texts=texts,
        )

        return import_stats

