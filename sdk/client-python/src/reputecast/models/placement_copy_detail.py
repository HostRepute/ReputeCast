from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_copy_detail_read_via import PlacementCopyDetailReadVia
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.placement_auth_check import PlacementAuthCheck
  from ..models.placement_copy_view import PlacementCopyView
  from ..models.placement_event import PlacementEvent





T = TypeVar("T", bound="PlacementCopyDetail")



@_attrs_define
class PlacementCopyDetail:
    """ 
        Attributes:
            copy (PlacementCopyView):
            events (list[PlacementEvent]):
            headers (str):
            key_headers (str):
            read_via (PlacementCopyDetailReadVia):
            test_status (str):
            dkim (PlacementAuthCheck | Unset):
            dmarc (PlacementAuthCheck | Unset):
            path (str | Unset):
            provider_message_id (str | Unset):
            spf (PlacementAuthCheck | Unset):
            wait_until (datetime.datetime | Unset):
     """

    copy: PlacementCopyView
    events: list[PlacementEvent]
    headers: str
    key_headers: str
    read_via: PlacementCopyDetailReadVia
    test_status: str
    dkim: PlacementAuthCheck | Unset = UNSET
    dmarc: PlacementAuthCheck | Unset = UNSET
    path: str | Unset = UNSET
    provider_message_id: str | Unset = UNSET
    spf: PlacementAuthCheck | Unset = UNSET
    wait_until: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_auth_check import PlacementAuthCheck # noqa: PLC0415
        from ..models.placement_copy_view import PlacementCopyView # noqa: PLC0415
        from ..models.placement_event import PlacementEvent # noqa: PLC0415
        copy = self.copy.to_dict()

        events = []
        for events_item_data in self.events:
            events_item = events_item_data.to_dict()
            events.append(events_item)



        headers = self.headers

        key_headers = self.key_headers

        read_via = self.read_via.value

        test_status = self.test_status

        dkim: dict[str, Any] | Unset = UNSET
        if not isinstance(self.dkim, Unset):
            dkim = self.dkim.to_dict()

        dmarc: dict[str, Any] | Unset = UNSET
        if not isinstance(self.dmarc, Unset):
            dmarc = self.dmarc.to_dict()

        path = self.path

        provider_message_id = self.provider_message_id

        spf: dict[str, Any] | Unset = UNSET
        if not isinstance(self.spf, Unset):
            spf = self.spf.to_dict()

        wait_until: str | Unset = UNSET
        if not isinstance(self.wait_until, Unset):
            wait_until = self.wait_until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "copy": copy,
            "events": events,
            "headers": headers,
            "keyHeaders": key_headers,
            "readVia": read_via,
            "testStatus": test_status,
        })
        if dkim is not UNSET:
            field_dict["dkim"] = dkim
        if dmarc is not UNSET:
            field_dict["dmarc"] = dmarc
        if path is not UNSET:
            field_dict["path"] = path
        if provider_message_id is not UNSET:
            field_dict["providerMessageId"] = provider_message_id
        if spf is not UNSET:
            field_dict["spf"] = spf
        if wait_until is not UNSET:
            field_dict["waitUntil"] = wait_until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_auth_check import PlacementAuthCheck # noqa: PLC0415
        from ..models.placement_copy_view import PlacementCopyView # noqa: PLC0415
        from ..models.placement_event import PlacementEvent # noqa: PLC0415
        d = dict(src_dict)
        copy = PlacementCopyView.from_dict(d.pop("copy"))




        events = []
        _events = d.pop("events")
        for events_item_data in (_events):
            events_item = PlacementEvent.from_dict(events_item_data)



            events.append(events_item)


        headers = d.pop("headers")

        key_headers = d.pop("keyHeaders")

        read_via = PlacementCopyDetailReadVia(d.pop("readVia"))




        test_status = d.pop("testStatus")

        _dkim = d.pop("dkim", UNSET)
        dkim: PlacementAuthCheck | Unset
        if isinstance(_dkim,  Unset):
            dkim = UNSET
        else:
            dkim = PlacementAuthCheck.from_dict(_dkim)




        _dmarc = d.pop("dmarc", UNSET)
        dmarc: PlacementAuthCheck | Unset
        if isinstance(_dmarc,  Unset):
            dmarc = UNSET
        else:
            dmarc = PlacementAuthCheck.from_dict(_dmarc)




        path = d.pop("path", UNSET)

        provider_message_id = d.pop("providerMessageId", UNSET)

        _spf = d.pop("spf", UNSET)
        spf: PlacementAuthCheck | Unset
        if isinstance(_spf,  Unset):
            spf = UNSET
        else:
            spf = PlacementAuthCheck.from_dict(_spf)




        _wait_until = d.pop("waitUntil", UNSET)
        wait_until: datetime.datetime | Unset
        if isinstance(_wait_until,  Unset):
            wait_until = UNSET
        else:
            wait_until = datetime.datetime.fromisoformat(_wait_until)




        placement_copy_detail = cls(
            copy=copy,
            events=events,
            headers=headers,
            key_headers=key_headers,
            read_via=read_via,
            test_status=test_status,
            dkim=dkim,
            dmarc=dmarc,
            path=path,
            provider_message_id=provider_message_id,
            spf=spf,
            wait_until=wait_until,
        )

        return placement_copy_detail

