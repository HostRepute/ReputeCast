from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.in_doubt_share import InDoubtShare





T = TypeVar("T", bound="LiveInDoubt")



@_attrs_define
class LiveInDoubt:
    """ 
        Attributes:
            count (int):
            providers (list[InDoubtShare]):
            safe (bool):
            since (datetime.datetime):
            waiting (bool):
            incident_id (str | Unset):
     """

    count: int
    providers: list[InDoubtShare]
    safe: bool
    since: datetime.datetime
    waiting: bool
    incident_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.in_doubt_share import InDoubtShare # noqa: PLC0415
        count = self.count

        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        safe = self.safe

        since = self.since.isoformat()

        waiting = self.waiting

        incident_id = self.incident_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "providers": providers,
            "safe": safe,
            "since": since,
            "waiting": waiting,
        })
        if incident_id is not UNSET:
            field_dict["incidentId"] = incident_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.in_doubt_share import InDoubtShare # noqa: PLC0415
        d = dict(src_dict)
        count = d.pop("count")

        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = InDoubtShare.from_dict(providers_item_data)



            providers.append(providers_item)


        safe = d.pop("safe")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        waiting = d.pop("waiting")

        incident_id = d.pop("incidentId", UNSET)

        live_in_doubt = cls(
            count=count,
            providers=providers,
            safe=safe,
            since=since,
            waiting=waiting,
            incident_id=incident_id,
        )

        return live_in_doubt

