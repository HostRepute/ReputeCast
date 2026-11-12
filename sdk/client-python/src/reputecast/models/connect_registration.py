from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_registration_status import ConnectRegistrationStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ConnectRegistration")



@_attrs_define
class ConnectRegistration:
    """ 
        Attributes:
            at (datetime.datetime):
            status (ConnectRegistrationStatus):
            detail (str | Unset):
     """

    at: datetime.datetime
    status: ConnectRegistrationStatus
    detail: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        status = self.status.value

        detail = self.detail


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "status": status,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        status = ConnectRegistrationStatus(d.pop("status"))




        detail = d.pop("detail", UNSET)

        connect_registration = cls(
            at=at,
            status=status,
            detail=detail,
        )

        return connect_registration

