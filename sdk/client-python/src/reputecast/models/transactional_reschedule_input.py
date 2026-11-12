from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalRescheduleInput")



@_attrs_define
class TransactionalRescheduleInput:
    """ 
        Attributes:
            send_at (datetime.datetime): The new send time, up to 72 hours from now
     """

    send_at: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "sendAt": send_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        send_at = datetime.datetime.fromisoformat(d.pop("sendAt"))




        transactional_reschedule_input = cls(
            send_at=send_at,
        )

        return transactional_reschedule_input

