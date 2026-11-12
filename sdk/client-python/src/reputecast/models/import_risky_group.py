from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_risky_group_key import ImportRiskyGroupKey
from typing import cast

if TYPE_CHECKING:
  from ..models.import_field_option import ImportFieldOption





T = TypeVar("T", bound="ImportRiskyGroup")



@_attrs_define
class ImportRiskyGroup:
    """ 
        Attributes:
            action (str):
            actions (list[ImportFieldOption]):
            agree (int):
            examples (str):
            key (ImportRiskyGroupKey):
            label (str):
            more (int):
            own (int):
            rows (int):
            why (str):
     """

    action: str
    actions: list[ImportFieldOption]
    agree: int
    examples: str
    key: ImportRiskyGroupKey
    label: str
    more: int
    own: int
    rows: int
    why: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_field_option import ImportFieldOption # noqa: PLC0415
        action = self.action

        actions = []
        for actions_item_data in self.actions:
            actions_item = actions_item_data.to_dict()
            actions.append(actions_item)



        agree = self.agree

        examples = self.examples

        key = self.key.value

        label = self.label

        more = self.more

        own = self.own

        rows = self.rows

        why = self.why


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
            "actions": actions,
            "agree": agree,
            "examples": examples,
            "key": key,
            "label": label,
            "more": more,
            "own": own,
            "rows": rows,
            "why": why,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_field_option import ImportFieldOption # noqa: PLC0415
        d = dict(src_dict)
        action = d.pop("action")

        actions = []
        _actions = d.pop("actions")
        for actions_item_data in (_actions):
            actions_item = ImportFieldOption.from_dict(actions_item_data)



            actions.append(actions_item)


        agree = d.pop("agree")

        examples = d.pop("examples")

        key = ImportRiskyGroupKey(d.pop("key"))




        label = d.pop("label")

        more = d.pop("more")

        own = d.pop("own")

        rows = d.pop("rows")

        why = d.pop("why")

        import_risky_group = cls(
            action=action,
            actions=actions,
            agree=agree,
            examples=examples,
            key=key,
            label=label,
            more=more,
            own=own,
            rows=rows,
            why=why,
        )

        return import_risky_group

