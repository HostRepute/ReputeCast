from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalFailover")



@_attrs_define
class TransactionalFailover:
    """ 
        Attributes:
            from_ (str):
            to (str):
            at (datetime.datetime | Unset):
     """

    from_: str
    to: str
    at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from_ = self.from_

        to = self.to

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "from": from_,
            "to": to,
        })
        if at is not UNSET:
            field_dict["at"] = at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        from_ = d.pop("from")

        to = d.pop("to")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        transactional_failover = cls(
            from_=from_,
            to=to,
            at=at,
        )

        return transactional_failover

