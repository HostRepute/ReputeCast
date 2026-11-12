from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.settings_provider_card_status import SettingsProviderCardStatus
from ..models.settings_provider_card_status_tone import SettingsProviderCardStatusTone
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.settings_provider_cost import SettingsProviderCost





T = TypeVar("T", bound="SettingsProviderCard")



@_attrs_define
class SettingsProviderCard:
    """ 
        Attributes:
            connected_at (datetime.datetime):
            connected_via (str):
            cost (SettingsProviderCost):
            id (str):
            identity (str):
            identity_label (str):
            name (str):
            status (SettingsProviderCardStatus):
            status_label (str):
            status_tone (SettingsProviderCardStatusTone):
            streams (str):
     """

    connected_at: datetime.datetime
    connected_via: str
    cost: SettingsProviderCost
    id: str
    identity: str
    identity_label: str
    name: str
    status: SettingsProviderCardStatus
    status_label: str
    status_tone: SettingsProviderCardStatusTone
    streams: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.settings_provider_cost import SettingsProviderCost # noqa: PLC0415
        connected_at = self.connected_at.isoformat()

        connected_via = self.connected_via

        cost = self.cost.to_dict()

        id = self.id

        identity = self.identity

        identity_label = self.identity_label

        name = self.name

        status = self.status.value

        status_label = self.status_label

        status_tone = self.status_tone.value

        streams = self.streams


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "connectedAt": connected_at,
            "connectedVia": connected_via,
            "cost": cost,
            "id": id,
            "identity": identity,
            "identityLabel": identity_label,
            "name": name,
            "status": status,
            "statusLabel": status_label,
            "statusTone": status_tone,
            "streams": streams,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.settings_provider_cost import SettingsProviderCost # noqa: PLC0415
        d = dict(src_dict)
        connected_at = datetime.datetime.fromisoformat(d.pop("connectedAt"))




        connected_via = d.pop("connectedVia")

        cost = SettingsProviderCost.from_dict(d.pop("cost"))




        id = d.pop("id")

        identity = d.pop("identity")

        identity_label = d.pop("identityLabel")

        name = d.pop("name")

        status = SettingsProviderCardStatus(d.pop("status"))




        status_label = d.pop("statusLabel")

        status_tone = SettingsProviderCardStatusTone(d.pop("statusTone"))




        streams = d.pop("streams")

        settings_provider_card = cls(
            connected_at=connected_at,
            connected_via=connected_via,
            cost=cost,
            id=id,
            identity=identity,
            identity_label=identity_label,
            name=name,
            status=status,
            status_label=status_label,
            status_tone=status_tone,
            streams=streams,
        )

        return settings_provider_card

