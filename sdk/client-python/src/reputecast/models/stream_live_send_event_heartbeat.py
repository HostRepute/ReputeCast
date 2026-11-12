from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
from typing import Literal, cast

if TYPE_CHECKING:
  from ..models.live_heartbeat import LiveHeartbeat





T = TypeVar("T", bound="StreamLiveSendEventHeartbeat")



@_attrs_define
class StreamLiveSendEventHeartbeat:
    """ 
        Attributes:
            data (LiveHeartbeat):
            event (Literal['heartbeat']): The event name.
            id (int | str | Unset): The event ID.
            retry (int | Unset): The retry time in milliseconds.
     """

    data: LiveHeartbeat
    event: Literal['heartbeat']
    id: int | str | Unset = UNSET
    retry: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.live_heartbeat import LiveHeartbeat # noqa: PLC0415
        data = self.data.to_dict()

        event = self.event

        id: int | str | Unset
        if isinstance(self.id, Unset):
            id = UNSET
        else:
            id = self.id

        retry = self.retry


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "data": data,
            "event": event,
        })
        if id is not UNSET:
            field_dict["id"] = id
        if retry is not UNSET:
            field_dict["retry"] = retry

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.live_heartbeat import LiveHeartbeat # noqa: PLC0415
        d = dict(src_dict)
        data = LiveHeartbeat.from_dict(d.pop("data"))




        event = cast(Literal['heartbeat'] , d.pop("event"))
        if event != 'heartbeat':
            raise ValueError(f"event must match const 'heartbeat', got '{event}'")

        def _parse_id(data: object) -> int | str | Unset:
            if isinstance(data, Unset):
                return data
            return cast(int | str | Unset, data)

        id = _parse_id(d.pop("id", UNSET))


        retry = d.pop("retry", UNSET)

        stream_live_send_event_heartbeat = cls(
            data=data,
            event=event,
            id=id,
            retry=retry,
        )


        stream_live_send_event_heartbeat.additional_properties = d
        return stream_live_send_event_heartbeat

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
