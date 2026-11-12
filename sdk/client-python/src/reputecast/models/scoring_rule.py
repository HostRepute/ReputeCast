from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.scoring_rule_kind import ScoringRuleKind
from ..models.scoring_rule_op import ScoringRuleOp
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.scoring_rule_cap import ScoringRuleCap





T = TypeVar("T", bound="ScoringRule")



@_attrs_define
class ScoringRule:
    """ 
        Attributes:
            cap (ScoringRuleCap):
            enabled (bool):
            id (str): Kept across versions; new rules get one when saved
            kind (ScoringRuleKind):
            points (int):
            contains (str | Unset):
            event (str | Unset):
            field (str | Unset):
            form_id (str | Unset):
            list_id (str | Unset):
            op (ScoringRuleOp | Unset):
            values (list[str] | Unset):
     """

    cap: ScoringRuleCap
    enabled: bool
    id: str
    kind: ScoringRuleKind
    points: int
    contains: str | Unset = UNSET
    event: str | Unset = UNSET
    field: str | Unset = UNSET
    form_id: str | Unset = UNSET
    list_id: str | Unset = UNSET
    op: ScoringRuleOp | Unset = UNSET
    values: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.scoring_rule_cap import ScoringRuleCap # noqa: PLC0415
        cap = self.cap.to_dict()

        enabled = self.enabled

        id = self.id

        kind = self.kind.value

        points = self.points

        contains = self.contains

        event = self.event

        field = self.field

        form_id = self.form_id

        list_id = self.list_id

        op: str | Unset = UNSET
        if not isinstance(self.op, Unset):
            op = self.op.value


        values: list[str] | Unset = UNSET
        if not isinstance(self.values, Unset):
            values = self.values




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cap": cap,
            "enabled": enabled,
            "id": id,
            "kind": kind,
            "points": points,
        })
        if contains is not UNSET:
            field_dict["contains"] = contains
        if event is not UNSET:
            field_dict["event"] = event
        if field is not UNSET:
            field_dict["field"] = field
        if form_id is not UNSET:
            field_dict["formId"] = form_id
        if list_id is not UNSET:
            field_dict["listId"] = list_id
        if op is not UNSET:
            field_dict["op"] = op
        if values is not UNSET:
            field_dict["values"] = values

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.scoring_rule_cap import ScoringRuleCap # noqa: PLC0415
        d = dict(src_dict)
        cap = ScoringRuleCap.from_dict(d.pop("cap"))




        enabled = d.pop("enabled")

        id = d.pop("id")

        kind = ScoringRuleKind(d.pop("kind"))




        points = d.pop("points")

        contains = d.pop("contains", UNSET)

        event = d.pop("event", UNSET)

        field = d.pop("field", UNSET)

        form_id = d.pop("formId", UNSET)

        list_id = d.pop("listId", UNSET)

        _op = d.pop("op", UNSET)
        op: ScoringRuleOp | Unset
        if isinstance(_op,  Unset):
            op = UNSET
        else:
            op = ScoringRuleOp(_op)




        values = cast(list[str], d.pop("values", UNSET))


        scoring_rule = cls(
            cap=cap,
            enabled=enabled,
            id=id,
            kind=kind,
            points=points,
            contains=contains,
            event=event,
            field=field,
            form_id=form_id,
            list_id=list_id,
            op=op,
            values=values,
        )

        return scoring_rule

