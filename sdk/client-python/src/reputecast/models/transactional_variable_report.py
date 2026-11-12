from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.variable_use import VariableUse





T = TypeVar("T", bound="TransactionalVariableReport")



@_attrs_define
class TransactionalVariableReport:
    """ 
        Attributes:
            undeclared (list[VariableUse]):
            unused (list[str]):
            used (list[VariableUse]):
     """

    undeclared: list[VariableUse]
    unused: list[str]
    used: list[VariableUse]





    def to_dict(self) -> dict[str, Any]:
        from ..models.variable_use import VariableUse # noqa: PLC0415
        undeclared = []
        for undeclared_item_data in self.undeclared:
            undeclared_item = undeclared_item_data.to_dict()
            undeclared.append(undeclared_item)



        unused = self.unused



        used = []
        for used_item_data in self.used:
            used_item = used_item_data.to_dict()
            used.append(used_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "undeclared": undeclared,
            "unused": unused,
            "used": used,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.variable_use import VariableUse # noqa: PLC0415
        d = dict(src_dict)
        undeclared = []
        _undeclared = d.pop("undeclared")
        for undeclared_item_data in (_undeclared):
            undeclared_item = VariableUse.from_dict(undeclared_item_data)



            undeclared.append(undeclared_item)


        unused = cast(list[str], d.pop("unused"))


        used = []
        _used = d.pop("used")
        for used_item_data in (_used):
            used_item = VariableUse.from_dict(used_item_data)



            used.append(used_item)


        transactional_variable_report = cls(
            undeclared=undeclared,
            unused=unused,
            used=used,
        )

        return transactional_variable_report

