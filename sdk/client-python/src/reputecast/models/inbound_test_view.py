from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="InboundTestView")



@_attrs_define
class InboundTestView:
    """ 
        Attributes:
            received_at (datetime.datetime | Unset):
            sent_at (datetime.datetime | Unset):
     """

    received_at: datetime.datetime | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        received_at: str | Unset = UNSET
        if not isinstance(self.received_at, Unset):
            received_at = self.received_at.isoformat()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if received_at is not UNSET:
            field_dict["receivedAt"] = received_at
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _received_at = d.pop("receivedAt", UNSET)
        received_at: datetime.datetime | Unset
        if isinstance(_received_at,  Unset):
            received_at = UNSET
        else:
            received_at = datetime.datetime.fromisoformat(_received_at)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        inbound_test_view = cls(
            received_at=received_at,
            sent_at=sent_at,
        )

        return inbound_test_view

