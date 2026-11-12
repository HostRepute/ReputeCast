from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_options_evidence import ImportOptionsEvidence
from ..models.import_options_on_conflict import ImportOptionsOnConflict
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.import_list_choice import ImportListChoice
  from ..models.import_risky_actions import ImportRiskyActions





T = TypeVar("T", bound="ImportOptions")



@_attrs_define
class ImportOptions:
    """ 
        Attributes:
            evidence (ImportOptionsEvidence):
            import_engagement (bool):
            import_tags (bool):
            import_templates (bool):
            lists (list[ImportListChoice]):
            on_conflict (ImportOptionsOnConflict):
            pull_suppressions (bool):
            record_history (bool):
            risky (ImportRiskyActions):
            skip_rows (list[int]):
            pace (int | Unset):
            tag (str | Unset):
     """

    evidence: ImportOptionsEvidence
    import_engagement: bool
    import_tags: bool
    import_templates: bool
    lists: list[ImportListChoice]
    on_conflict: ImportOptionsOnConflict
    pull_suppressions: bool
    record_history: bool
    risky: ImportRiskyActions
    skip_rows: list[int]
    pace: int | Unset = UNSET
    tag: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_list_choice import ImportListChoice # noqa: PLC0415
        from ..models.import_risky_actions import ImportRiskyActions # noqa: PLC0415
        evidence = self.evidence.value

        import_engagement = self.import_engagement

        import_tags = self.import_tags

        import_templates = self.import_templates

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        on_conflict = self.on_conflict.value

        pull_suppressions = self.pull_suppressions

        record_history = self.record_history

        risky = self.risky.to_dict()

        skip_rows = self.skip_rows



        pace = self.pace

        tag = self.tag


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "evidence": evidence,
            "importEngagement": import_engagement,
            "importTags": import_tags,
            "importTemplates": import_templates,
            "lists": lists,
            "onConflict": on_conflict,
            "pullSuppressions": pull_suppressions,
            "recordHistory": record_history,
            "risky": risky,
            "skipRows": skip_rows,
        })
        if pace is not UNSET:
            field_dict["pace"] = pace
        if tag is not UNSET:
            field_dict["tag"] = tag

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_list_choice import ImportListChoice # noqa: PLC0415
        from ..models.import_risky_actions import ImportRiskyActions # noqa: PLC0415
        d = dict(src_dict)
        evidence = ImportOptionsEvidence(d.pop("evidence"))




        import_engagement = d.pop("importEngagement")

        import_tags = d.pop("importTags")

        import_templates = d.pop("importTemplates")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = ImportListChoice.from_dict(lists_item_data)



            lists.append(lists_item)


        on_conflict = ImportOptionsOnConflict(d.pop("onConflict"))




        pull_suppressions = d.pop("pullSuppressions")

        record_history = d.pop("recordHistory")

        risky = ImportRiskyActions.from_dict(d.pop("risky"))




        skip_rows = cast(list[int], d.pop("skipRows"))


        pace = d.pop("pace", UNSET)

        tag = d.pop("tag", UNSET)

        import_options = cls(
            evidence=evidence,
            import_engagement=import_engagement,
            import_tags=import_tags,
            import_templates=import_templates,
            lists=lists,
            on_conflict=on_conflict,
            pull_suppressions=pull_suppressions,
            record_history=record_history,
            risky=risky,
            skip_rows=skip_rows,
            pace=pace,
            tag=tag,
        )

        return import_options

