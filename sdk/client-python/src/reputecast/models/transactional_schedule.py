from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_schedule_status import TransactionalScheduleStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalSchedule")



@_attrs_define
class TransactionalSchedule:
    """ 
        Attributes:
            id (str):
            status (TransactionalScheduleStatus):
            send_at (datetime.datetime | Unset):
     """

    id: str
    status: TransactionalScheduleStatus
    send_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        status = self.status.value

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "status": status,
        })
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        status = TransactionalScheduleStatus(d.pop("status"))




        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        transactional_schedule = cls(
            id=id,
            status=status,
            send_at=send_at,
        )

        return transactional_schedule

