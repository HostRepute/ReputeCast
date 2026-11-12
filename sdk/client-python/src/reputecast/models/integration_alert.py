from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.integration_reason import IntegrationReason





T = TypeVar("T", bound="IntegrationAlert")



@_attrs_define
class IntegrationAlert:
    """ 
        Attributes:
            entry (str):
            reason (IntegrationReason):
            address (str | Unset):
            connection_id (str | Unset):
     """

    entry: str
    reason: IntegrationReason
    address: str | Unset = UNSET
    connection_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_reason import IntegrationReason # noqa: PLC0415
        entry = self.entry

        reason = self.reason.to_dict()

        address = self.address

        connection_id = self.connection_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "entry": entry,
            "reason": reason,
        })
        if address is not UNSET:
            field_dict["address"] = address
        if connection_id is not UNSET:
            field_dict["connectionId"] = connection_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_reason import IntegrationReason # noqa: PLC0415
        d = dict(src_dict)
        entry = d.pop("entry")

        reason = IntegrationReason.from_dict(d.pop("reason"))




        address = d.pop("address", UNSET)

        connection_id = d.pop("connectionId", UNSET)

        integration_alert = cls(
            entry=entry,
            reason=reason,
            address=address,
            connection_id=connection_id,
        )

        return integration_alert

