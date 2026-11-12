from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_status_status import IntegrationStatusStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.integration_first_call import IntegrationFirstCall





T = TypeVar("T", bound="IntegrationStatus")



@_attrs_define
class IntegrationStatus:
    """ 
        Attributes:
            status (IntegrationStatusStatus):
            first_call (IntegrationFirstCall | Unset):
            key_created_at (datetime.datetime | Unset):
            reported (datetime.datetime | Unset):
     """

    status: IntegrationStatusStatus
    first_call: IntegrationFirstCall | Unset = UNSET
    key_created_at: datetime.datetime | Unset = UNSET
    reported: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_first_call import IntegrationFirstCall # noqa: PLC0415
        status = self.status.value

        first_call: dict[str, Any] | Unset = UNSET
        if not isinstance(self.first_call, Unset):
            first_call = self.first_call.to_dict()

        key_created_at: str | Unset = UNSET
        if not isinstance(self.key_created_at, Unset):
            key_created_at = self.key_created_at.isoformat()

        reported: str | Unset = UNSET
        if not isinstance(self.reported, Unset):
            reported = self.reported.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "status": status,
        })
        if first_call is not UNSET:
            field_dict["firstCall"] = first_call
        if key_created_at is not UNSET:
            field_dict["keyCreatedAt"] = key_created_at
        if reported is not UNSET:
            field_dict["reported"] = reported

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_first_call import IntegrationFirstCall # noqa: PLC0415
        d = dict(src_dict)
        status = IntegrationStatusStatus(d.pop("status"))




        _first_call = d.pop("firstCall", UNSET)
        first_call: IntegrationFirstCall | Unset
        if isinstance(_first_call,  Unset):
            first_call = UNSET
        else:
            first_call = IntegrationFirstCall.from_dict(_first_call)




        _key_created_at = d.pop("keyCreatedAt", UNSET)
        key_created_at: datetime.datetime | Unset
        if isinstance(_key_created_at,  Unset):
            key_created_at = UNSET
        else:
            key_created_at = datetime.datetime.fromisoformat(_key_created_at)




        _reported = d.pop("reported", UNSET)
        reported: datetime.datetime | Unset
        if isinstance(_reported,  Unset):
            reported = UNSET
        else:
            reported = datetime.datetime.fromisoformat(_reported)




        integration_status = cls(
            status=status,
            first_call=first_call,
            key_created_at=key_created_at,
            reported=reported,
        )

        return integration_status

