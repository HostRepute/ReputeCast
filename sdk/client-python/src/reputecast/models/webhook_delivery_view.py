from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.webhook_delivery_view_state import WebhookDeliveryViewState
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WebhookDeliveryView")



@_attrs_define
class WebhookDeliveryView:
    """ 
        Attributes:
            attempts (int):
            created_at (datetime.datetime):
            endpoint (str): Host and path, e.g. api.acme.io/hooks/reputecast
            event (str):
            id (str):
            max_attempts (int):
            state (WebhookDeliveryViewState):
            webhook_id (str):
            backoff_minutes (int | Unset):
            duration_ms (int | Unset):
            error (str | Unset):
            last_attempt_at (datetime.datetime | Unset):
            next_attempt_at (datetime.datetime | Unset):
            status_code (int | Unset):
            status_text (str | Unset): Bad Gateway
            subtype (str | Unset):
     """

    attempts: int
    created_at: datetime.datetime
    endpoint: str
    event: str
    id: str
    max_attempts: int
    state: WebhookDeliveryViewState
    webhook_id: str
    backoff_minutes: int | Unset = UNSET
    duration_ms: int | Unset = UNSET
    error: str | Unset = UNSET
    last_attempt_at: datetime.datetime | Unset = UNSET
    next_attempt_at: datetime.datetime | Unset = UNSET
    status_code: int | Unset = UNSET
    status_text: str | Unset = UNSET
    subtype: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        attempts = self.attempts

        created_at = self.created_at.isoformat()

        endpoint = self.endpoint

        event = self.event

        id = self.id

        max_attempts = self.max_attempts

        state = self.state.value

        webhook_id = self.webhook_id

        backoff_minutes = self.backoff_minutes

        duration_ms = self.duration_ms

        error = self.error

        last_attempt_at: str | Unset = UNSET
        if not isinstance(self.last_attempt_at, Unset):
            last_attempt_at = self.last_attempt_at.isoformat()

        next_attempt_at: str | Unset = UNSET
        if not isinstance(self.next_attempt_at, Unset):
            next_attempt_at = self.next_attempt_at.isoformat()

        status_code = self.status_code

        status_text = self.status_text

        subtype = self.subtype


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "attempts": attempts,
            "createdAt": created_at,
            "endpoint": endpoint,
            "event": event,
            "id": id,
            "maxAttempts": max_attempts,
            "state": state,
            "webhookId": webhook_id,
        })
        if backoff_minutes is not UNSET:
            field_dict["backoffMinutes"] = backoff_minutes
        if duration_ms is not UNSET:
            field_dict["durationMs"] = duration_ms
        if error is not UNSET:
            field_dict["error"] = error
        if last_attempt_at is not UNSET:
            field_dict["lastAttemptAt"] = last_attempt_at
        if next_attempt_at is not UNSET:
            field_dict["nextAttemptAt"] = next_attempt_at
        if status_code is not UNSET:
            field_dict["statusCode"] = status_code
        if status_text is not UNSET:
            field_dict["statusText"] = status_text
        if subtype is not UNSET:
            field_dict["subtype"] = subtype

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        attempts = d.pop("attempts")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        endpoint = d.pop("endpoint")

        event = d.pop("event")

        id = d.pop("id")

        max_attempts = d.pop("maxAttempts")

        state = WebhookDeliveryViewState(d.pop("state"))




        webhook_id = d.pop("webhookId")

        backoff_minutes = d.pop("backoffMinutes", UNSET)

        duration_ms = d.pop("durationMs", UNSET)

        error = d.pop("error", UNSET)

        _last_attempt_at = d.pop("lastAttemptAt", UNSET)
        last_attempt_at: datetime.datetime | Unset
        if isinstance(_last_attempt_at,  Unset):
            last_attempt_at = UNSET
        else:
            last_attempt_at = datetime.datetime.fromisoformat(_last_attempt_at)




        _next_attempt_at = d.pop("nextAttemptAt", UNSET)
        next_attempt_at: datetime.datetime | Unset
        if isinstance(_next_attempt_at,  Unset):
            next_attempt_at = UNSET
        else:
            next_attempt_at = datetime.datetime.fromisoformat(_next_attempt_at)




        status_code = d.pop("statusCode", UNSET)

        status_text = d.pop("statusText", UNSET)

        subtype = d.pop("subtype", UNSET)

        webhook_delivery_view = cls(
            attempts=attempts,
            created_at=created_at,
            endpoint=endpoint,
            event=event,
            id=id,
            max_attempts=max_attempts,
            state=state,
            webhook_id=webhook_id,
            backoff_minutes=backoff_minutes,
            duration_ms=duration_ms,
            error=error,
            last_attempt_at=last_attempt_at,
            next_attempt_at=next_attempt_at,
            status_code=status_code,
            status_text=status_text,
            subtype=subtype,
        )

        return webhook_delivery_view

