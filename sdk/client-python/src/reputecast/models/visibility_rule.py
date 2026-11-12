from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.visibility_rule_match import VisibilityRuleMatch
from ..models.visibility_rule_show import VisibilityRuleShow
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.visibility_condition import VisibilityCondition





T = TypeVar("T", bound="VisibilityRule")



@_attrs_define
class VisibilityRule:
    """ 
        Attributes:
            conditions (list[VisibilityCondition]):
            id (str):
            show (VisibilityRuleShow):
            match (VisibilityRuleMatch | Unset):
     """

    conditions: list[VisibilityCondition]
    id: str
    show: VisibilityRuleShow
    match: VisibilityRuleMatch | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.visibility_condition import VisibilityCondition # noqa: PLC0415
        conditions = []
        for conditions_item_data in self.conditions:
            conditions_item = conditions_item_data.to_dict()
            conditions.append(conditions_item)



        id = self.id

        show = self.show.value

        match: str | Unset = UNSET
        if not isinstance(self.match, Unset):
            match = self.match.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "conditions": conditions,
            "id": id,
            "show": show,
        })
        if match is not UNSET:
            field_dict["match"] = match

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.visibility_condition import VisibilityCondition # noqa: PLC0415
        d = dict(src_dict)
        conditions = []
        _conditions = d.pop("conditions")
        for conditions_item_data in (_conditions):
            conditions_item = VisibilityCondition.from_dict(conditions_item_data)



            conditions.append(conditions_item)


        id = d.pop("id")

        show = VisibilityRuleShow(d.pop("show"))




        _match = d.pop("match", UNSET)
        match: VisibilityRuleMatch | Unset
        if isinstance(_match,  Unset):
            match = UNSET
        else:
            match = VisibilityRuleMatch(_match)




        visibility_rule = cls(
            conditions=conditions,
            id=id,
            show=show,
            match=match,
        )

        return visibility_rule

