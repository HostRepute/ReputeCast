from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_account_quality import WhatsAppAccountQuality
from ..models.whats_app_account_status import WhatsAppAccountStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppAccount")



@_attrs_define
class WhatsAppAccount:
    """ 
        Attributes:
            business_hint (str):
            business_name (str):
            connected_at (datetime.datetime):
            connected_by (str):
            demo (bool):
            display_name (str):
            held (list[str]):
            held_count (int):
            id (str):
            messaging_limit (int):
            name_status (str):
            number (str):
            old_address (bool):
            other_account_events (int):
            phone_status (str):
            quality (WhatsAppAccountQuality):
            rejected_deliveries (int):
            status (WhatsAppAccountStatus):
            webhook_url (str):
            wrong_address_deliveries (int):
            checked_at (datetime.datetime | Unset):
            last_event_at (datetime.datetime | Unset):
            limit_changed_at (datetime.datetime | Unset):
            other_account_at (datetime.datetime | Unset):
            previous_limit (int | Unset):
            previous_quality (str | Unset):
            quality_changed_at (datetime.datetime | Unset):
            rejected_at (datetime.datetime | Unset):
            status_detail (str | Unset):
            status_since (datetime.datetime | Unset):
            templates_synced_at (datetime.datetime | Unset):
            verified_at (datetime.datetime | Unset):
            verify_token (str | Unset):
            wrong_address_at (datetime.datetime | Unset):
     """

    business_hint: str
    business_name: str
    connected_at: datetime.datetime
    connected_by: str
    demo: bool
    display_name: str
    held: list[str]
    held_count: int
    id: str
    messaging_limit: int
    name_status: str
    number: str
    old_address: bool
    other_account_events: int
    phone_status: str
    quality: WhatsAppAccountQuality
    rejected_deliveries: int
    status: WhatsAppAccountStatus
    webhook_url: str
    wrong_address_deliveries: int
    checked_at: datetime.datetime | Unset = UNSET
    last_event_at: datetime.datetime | Unset = UNSET
    limit_changed_at: datetime.datetime | Unset = UNSET
    other_account_at: datetime.datetime | Unset = UNSET
    previous_limit: int | Unset = UNSET
    previous_quality: str | Unset = UNSET
    quality_changed_at: datetime.datetime | Unset = UNSET
    rejected_at: datetime.datetime | Unset = UNSET
    status_detail: str | Unset = UNSET
    status_since: datetime.datetime | Unset = UNSET
    templates_synced_at: datetime.datetime | Unset = UNSET
    verified_at: datetime.datetime | Unset = UNSET
    verify_token: str | Unset = UNSET
    wrong_address_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        business_hint = self.business_hint

        business_name = self.business_name

        connected_at = self.connected_at.isoformat()

        connected_by = self.connected_by

        demo = self.demo

        display_name = self.display_name

        held = self.held



        held_count = self.held_count

        id = self.id

        messaging_limit = self.messaging_limit

        name_status = self.name_status

        number = self.number

        old_address = self.old_address

        other_account_events = self.other_account_events

        phone_status = self.phone_status

        quality = self.quality.value

        rejected_deliveries = self.rejected_deliveries

        status = self.status.value

        webhook_url = self.webhook_url

        wrong_address_deliveries = self.wrong_address_deliveries

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        last_event_at: str | Unset = UNSET
        if not isinstance(self.last_event_at, Unset):
            last_event_at = self.last_event_at.isoformat()

        limit_changed_at: str | Unset = UNSET
        if not isinstance(self.limit_changed_at, Unset):
            limit_changed_at = self.limit_changed_at.isoformat()

        other_account_at: str | Unset = UNSET
        if not isinstance(self.other_account_at, Unset):
            other_account_at = self.other_account_at.isoformat()

        previous_limit = self.previous_limit

        previous_quality = self.previous_quality

        quality_changed_at: str | Unset = UNSET
        if not isinstance(self.quality_changed_at, Unset):
            quality_changed_at = self.quality_changed_at.isoformat()

        rejected_at: str | Unset = UNSET
        if not isinstance(self.rejected_at, Unset):
            rejected_at = self.rejected_at.isoformat()

        status_detail = self.status_detail

        status_since: str | Unset = UNSET
        if not isinstance(self.status_since, Unset):
            status_since = self.status_since.isoformat()

        templates_synced_at: str | Unset = UNSET
        if not isinstance(self.templates_synced_at, Unset):
            templates_synced_at = self.templates_synced_at.isoformat()

        verified_at: str | Unset = UNSET
        if not isinstance(self.verified_at, Unset):
            verified_at = self.verified_at.isoformat()

        verify_token = self.verify_token

        wrong_address_at: str | Unset = UNSET
        if not isinstance(self.wrong_address_at, Unset):
            wrong_address_at = self.wrong_address_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "businessHint": business_hint,
            "businessName": business_name,
            "connectedAt": connected_at,
            "connectedBy": connected_by,
            "demo": demo,
            "displayName": display_name,
            "held": held,
            "heldCount": held_count,
            "id": id,
            "messagingLimit": messaging_limit,
            "nameStatus": name_status,
            "number": number,
            "oldAddress": old_address,
            "otherAccountEvents": other_account_events,
            "phoneStatus": phone_status,
            "quality": quality,
            "rejectedDeliveries": rejected_deliveries,
            "status": status,
            "webhookUrl": webhook_url,
            "wrongAddressDeliveries": wrong_address_deliveries,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if last_event_at is not UNSET:
            field_dict["lastEventAt"] = last_event_at
        if limit_changed_at is not UNSET:
            field_dict["limitChangedAt"] = limit_changed_at
        if other_account_at is not UNSET:
            field_dict["otherAccountAt"] = other_account_at
        if previous_limit is not UNSET:
            field_dict["previousLimit"] = previous_limit
        if previous_quality is not UNSET:
            field_dict["previousQuality"] = previous_quality
        if quality_changed_at is not UNSET:
            field_dict["qualityChangedAt"] = quality_changed_at
        if rejected_at is not UNSET:
            field_dict["rejectedAt"] = rejected_at
        if status_detail is not UNSET:
            field_dict["statusDetail"] = status_detail
        if status_since is not UNSET:
            field_dict["statusSince"] = status_since
        if templates_synced_at is not UNSET:
            field_dict["templatesSyncedAt"] = templates_synced_at
        if verified_at is not UNSET:
            field_dict["verifiedAt"] = verified_at
        if verify_token is not UNSET:
            field_dict["verifyToken"] = verify_token
        if wrong_address_at is not UNSET:
            field_dict["wrongAddressAt"] = wrong_address_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        business_hint = d.pop("businessHint")

        business_name = d.pop("businessName")

        connected_at = datetime.datetime.fromisoformat(d.pop("connectedAt"))




        connected_by = d.pop("connectedBy")

        demo = d.pop("demo")

        display_name = d.pop("displayName")

        held = cast(list[str], d.pop("held"))


        held_count = d.pop("heldCount")

        id = d.pop("id")

        messaging_limit = d.pop("messagingLimit")

        name_status = d.pop("nameStatus")

        number = d.pop("number")

        old_address = d.pop("oldAddress")

        other_account_events = d.pop("otherAccountEvents")

        phone_status = d.pop("phoneStatus")

        quality = WhatsAppAccountQuality(d.pop("quality"))




        rejected_deliveries = d.pop("rejectedDeliveries")

        status = WhatsAppAccountStatus(d.pop("status"))




        webhook_url = d.pop("webhookUrl")

        wrong_address_deliveries = d.pop("wrongAddressDeliveries")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        _last_event_at = d.pop("lastEventAt", UNSET)
        last_event_at: datetime.datetime | Unset
        if isinstance(_last_event_at,  Unset):
            last_event_at = UNSET
        else:
            last_event_at = datetime.datetime.fromisoformat(_last_event_at)




        _limit_changed_at = d.pop("limitChangedAt", UNSET)
        limit_changed_at: datetime.datetime | Unset
        if isinstance(_limit_changed_at,  Unset):
            limit_changed_at = UNSET
        else:
            limit_changed_at = datetime.datetime.fromisoformat(_limit_changed_at)




        _other_account_at = d.pop("otherAccountAt", UNSET)
        other_account_at: datetime.datetime | Unset
        if isinstance(_other_account_at,  Unset):
            other_account_at = UNSET
        else:
            other_account_at = datetime.datetime.fromisoformat(_other_account_at)




        previous_limit = d.pop("previousLimit", UNSET)

        previous_quality = d.pop("previousQuality", UNSET)

        _quality_changed_at = d.pop("qualityChangedAt", UNSET)
        quality_changed_at: datetime.datetime | Unset
        if isinstance(_quality_changed_at,  Unset):
            quality_changed_at = UNSET
        else:
            quality_changed_at = datetime.datetime.fromisoformat(_quality_changed_at)




        _rejected_at = d.pop("rejectedAt", UNSET)
        rejected_at: datetime.datetime | Unset
        if isinstance(_rejected_at,  Unset):
            rejected_at = UNSET
        else:
            rejected_at = datetime.datetime.fromisoformat(_rejected_at)




        status_detail = d.pop("statusDetail", UNSET)

        _status_since = d.pop("statusSince", UNSET)
        status_since: datetime.datetime | Unset
        if isinstance(_status_since,  Unset):
            status_since = UNSET
        else:
            status_since = datetime.datetime.fromisoformat(_status_since)




        _templates_synced_at = d.pop("templatesSyncedAt", UNSET)
        templates_synced_at: datetime.datetime | Unset
        if isinstance(_templates_synced_at,  Unset):
            templates_synced_at = UNSET
        else:
            templates_synced_at = datetime.datetime.fromisoformat(_templates_synced_at)




        _verified_at = d.pop("verifiedAt", UNSET)
        verified_at: datetime.datetime | Unset
        if isinstance(_verified_at,  Unset):
            verified_at = UNSET
        else:
            verified_at = datetime.datetime.fromisoformat(_verified_at)




        verify_token = d.pop("verifyToken", UNSET)

        _wrong_address_at = d.pop("wrongAddressAt", UNSET)
        wrong_address_at: datetime.datetime | Unset
        if isinstance(_wrong_address_at,  Unset):
            wrong_address_at = UNSET
        else:
            wrong_address_at = datetime.datetime.fromisoformat(_wrong_address_at)




        whats_app_account = cls(
            business_hint=business_hint,
            business_name=business_name,
            connected_at=connected_at,
            connected_by=connected_by,
            demo=demo,
            display_name=display_name,
            held=held,
            held_count=held_count,
            id=id,
            messaging_limit=messaging_limit,
            name_status=name_status,
            number=number,
            old_address=old_address,
            other_account_events=other_account_events,
            phone_status=phone_status,
            quality=quality,
            rejected_deliveries=rejected_deliveries,
            status=status,
            webhook_url=webhook_url,
            wrong_address_deliveries=wrong_address_deliveries,
            checked_at=checked_at,
            last_event_at=last_event_at,
            limit_changed_at=limit_changed_at,
            other_account_at=other_account_at,
            previous_limit=previous_limit,
            previous_quality=previous_quality,
            quality_changed_at=quality_changed_at,
            rejected_at=rejected_at,
            status_detail=status_detail,
            status_since=status_since,
            templates_synced_at=templates_synced_at,
            verified_at=verified_at,
            verify_token=verify_token,
            wrong_address_at=wrong_address_at,
        )

        return whats_app_account

