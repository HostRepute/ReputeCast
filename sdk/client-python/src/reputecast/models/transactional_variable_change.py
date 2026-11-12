from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_variable_change_change import TransactionalVariableChangeChange
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_variable import TransactionalVariable





T = TypeVar("T", bound="TransactionalVariableChange")



@_attrs_define
class TransactionalVariableChange:
    """ 
        Attributes:
            change (TransactionalVariableChangeChange):
            name (str):
            summary (str):  Example: date · required.
            after (TransactionalVariable | Unset):
            before (TransactionalVariable | Unset):
     """

    change: TransactionalVariableChangeChange
    name: str
    summary: str
    after: TransactionalVariable | Unset = UNSET
    before: TransactionalVariable | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        change = self.change.value

        name = self.name

        summary = self.summary

        after: dict[str, Any] | Unset = UNSET
        if not isinstance(self.after, Unset):
            after = self.after.to_dict()

        before: dict[str, Any] | Unset = UNSET
        if not isinstance(self.before, Unset):
            before = self.before.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "change": change,
            "name": name,
            "summary": summary,
        })
        if after is not UNSET:
            field_dict["after"] = after
        if before is not UNSET:
            field_dict["before"] = before

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        d = dict(src_dict)
        change = TransactionalVariableChangeChange(d.pop("change"))




        name = d.pop("name")

        summary = d.pop("summary")

        _after = d.pop("after", UNSET)
        after: TransactionalVariable | Unset
        if isinstance(_after,  Unset):
            after = UNSET
        else:
            after = TransactionalVariable.from_dict(_after)




        _before = d.pop("before", UNSET)
        before: TransactionalVariable | Unset
        if isinstance(_before,  Unset):
            before = UNSET
        else:
            before = TransactionalVariable.from_dict(_before)




        transactional_variable_change = cls(
            change=change,
            name=name,
            summary=summary,
            after=after,
            before=before,
        )

        return transactional_variable_change

