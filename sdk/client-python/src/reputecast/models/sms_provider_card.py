from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_provider_card_opt_out_sync import SmsProviderCardOptOutSync
from ..models.sms_provider_card_status import SmsProviderCardStatus
from ..models.sms_provider_card_webhooks import SmsProviderCardWebhooks
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.badge import Badge





T = TypeVar("T", bound="SmsProviderCard")



@_attrs_define
class SmsProviderCard:
    """ 
        Attributes:
            account_hint (str):
            account_label (str):
            adapter (str):
            badges (list[Badge]):
            connected_at (datetime.datetime):
            connected_by (str):
            console (str):
            countries (list[str]):
            demo (bool):
            failed_pct (float):
            filtered_pct (float):
            held (list[str]):
            held_texts (int):
            id (str):
            live (bool):
            name (str):
            opt_out_sync (SmsProviderCardOptOutSync):
            pause_failed (float):
            pause_filtered (float):
            price_countries (list[str]):
            price_max (float):
            price_min (float):
            receipts_since (int):
            secret_labels (list[str]):
            sender_count (int):
            sender_id_countries (int):
            sender_ids (list[str]):
            sender_kinds (list[str]):
            sent_since (int):
            spend (float):
            status (SmsProviderCardStatus):
            test_mode (bool):
            told (list[str]):
            webhook_url (str):
            webhooks (SmsProviderCardWebhooks):
            answer (str | Unset):
            detail (str | Unset):
            last_checked_at (datetime.datetime | Unset):
            last_receipt_at (datetime.datetime | Unset):
            since (datetime.datetime | Unset):
     """

    account_hint: str
    account_label: str
    adapter: str
    badges: list[Badge]
    connected_at: datetime.datetime
    connected_by: str
    console: str
    countries: list[str]
    demo: bool
    failed_pct: float
    filtered_pct: float
    held: list[str]
    held_texts: int
    id: str
    live: bool
    name: str
    opt_out_sync: SmsProviderCardOptOutSync
    pause_failed: float
    pause_filtered: float
    price_countries: list[str]
    price_max: float
    price_min: float
    receipts_since: int
    secret_labels: list[str]
    sender_count: int
    sender_id_countries: int
    sender_ids: list[str]
    sender_kinds: list[str]
    sent_since: int
    spend: float
    status: SmsProviderCardStatus
    test_mode: bool
    told: list[str]
    webhook_url: str
    webhooks: SmsProviderCardWebhooks
    answer: str | Unset = UNSET
    detail: str | Unset = UNSET
    last_checked_at: datetime.datetime | Unset = UNSET
    last_receipt_at: datetime.datetime | Unset = UNSET
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.badge import Badge # noqa: PLC0415
        account_hint = self.account_hint

        account_label = self.account_label

        adapter = self.adapter

        badges = []
        for badges_item_data in self.badges:
            badges_item = badges_item_data.to_dict()
            badges.append(badges_item)



        connected_at = self.connected_at.isoformat()

        connected_by = self.connected_by

        console = self.console

        countries = self.countries



        demo = self.demo

        failed_pct = self.failed_pct

        filtered_pct = self.filtered_pct

        held = self.held



        held_texts = self.held_texts

        id = self.id

        live = self.live

        name = self.name

        opt_out_sync = self.opt_out_sync.value

        pause_failed = self.pause_failed

        pause_filtered = self.pause_filtered

        price_countries = self.price_countries



        price_max = self.price_max

        price_min = self.price_min

        receipts_since = self.receipts_since

        secret_labels = self.secret_labels



        sender_count = self.sender_count

        sender_id_countries = self.sender_id_countries

        sender_ids = self.sender_ids



        sender_kinds = self.sender_kinds



        sent_since = self.sent_since

        spend = self.spend

        status = self.status.value

        test_mode = self.test_mode

        told = self.told



        webhook_url = self.webhook_url

        webhooks = self.webhooks.value

        answer = self.answer

        detail = self.detail

        last_checked_at: str | Unset = UNSET
        if not isinstance(self.last_checked_at, Unset):
            last_checked_at = self.last_checked_at.isoformat()

        last_receipt_at: str | Unset = UNSET
        if not isinstance(self.last_receipt_at, Unset):
            last_receipt_at = self.last_receipt_at.isoformat()

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accountHint": account_hint,
            "accountLabel": account_label,
            "adapter": adapter,
            "badges": badges,
            "connectedAt": connected_at,
            "connectedBy": connected_by,
            "console": console,
            "countries": countries,
            "demo": demo,
            "failedPct": failed_pct,
            "filteredPct": filtered_pct,
            "held": held,
            "heldTexts": held_texts,
            "id": id,
            "live": live,
            "name": name,
            "optOutSync": opt_out_sync,
            "pauseFailed": pause_failed,
            "pauseFiltered": pause_filtered,
            "priceCountries": price_countries,
            "priceMax": price_max,
            "priceMin": price_min,
            "receiptsSince": receipts_since,
            "secretLabels": secret_labels,
            "senderCount": sender_count,
            "senderIdCountries": sender_id_countries,
            "senderIds": sender_ids,
            "senderKinds": sender_kinds,
            "sentSince": sent_since,
            "spend": spend,
            "status": status,
            "testMode": test_mode,
            "told": told,
            "webhookUrl": webhook_url,
            "webhooks": webhooks,
        })
        if answer is not UNSET:
            field_dict["answer"] = answer
        if detail is not UNSET:
            field_dict["detail"] = detail
        if last_checked_at is not UNSET:
            field_dict["lastCheckedAt"] = last_checked_at
        if last_receipt_at is not UNSET:
            field_dict["lastReceiptAt"] = last_receipt_at
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.badge import Badge # noqa: PLC0415
        d = dict(src_dict)
        account_hint = d.pop("accountHint")

        account_label = d.pop("accountLabel")

        adapter = d.pop("adapter")

        badges = []
        _badges = d.pop("badges")
        for badges_item_data in (_badges):
            badges_item = Badge.from_dict(badges_item_data)



            badges.append(badges_item)


        connected_at = datetime.datetime.fromisoformat(d.pop("connectedAt"))




        connected_by = d.pop("connectedBy")

        console = d.pop("console")

        countries = cast(list[str], d.pop("countries"))


        demo = d.pop("demo")

        failed_pct = d.pop("failedPct")

        filtered_pct = d.pop("filteredPct")

        held = cast(list[str], d.pop("held"))


        held_texts = d.pop("heldTexts")

        id = d.pop("id")

        live = d.pop("live")

        name = d.pop("name")

        opt_out_sync = SmsProviderCardOptOutSync(d.pop("optOutSync"))




        pause_failed = d.pop("pauseFailed")

        pause_filtered = d.pop("pauseFiltered")

        price_countries = cast(list[str], d.pop("priceCountries"))


        price_max = d.pop("priceMax")

        price_min = d.pop("priceMin")

        receipts_since = d.pop("receiptsSince")

        secret_labels = cast(list[str], d.pop("secretLabels"))


        sender_count = d.pop("senderCount")

        sender_id_countries = d.pop("senderIdCountries")

        sender_ids = cast(list[str], d.pop("senderIds"))


        sender_kinds = cast(list[str], d.pop("senderKinds"))


        sent_since = d.pop("sentSince")

        spend = d.pop("spend")

        status = SmsProviderCardStatus(d.pop("status"))




        test_mode = d.pop("testMode")

        told = cast(list[str], d.pop("told"))


        webhook_url = d.pop("webhookUrl")

        webhooks = SmsProviderCardWebhooks(d.pop("webhooks"))




        answer = d.pop("answer", UNSET)

        detail = d.pop("detail", UNSET)

        _last_checked_at = d.pop("lastCheckedAt", UNSET)
        last_checked_at: datetime.datetime | Unset
        if isinstance(_last_checked_at,  Unset):
            last_checked_at = UNSET
        else:
            last_checked_at = datetime.datetime.fromisoformat(_last_checked_at)




        _last_receipt_at = d.pop("lastReceiptAt", UNSET)
        last_receipt_at: datetime.datetime | Unset
        if isinstance(_last_receipt_at,  Unset):
            last_receipt_at = UNSET
        else:
            last_receipt_at = datetime.datetime.fromisoformat(_last_receipt_at)




        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        sms_provider_card = cls(
            account_hint=account_hint,
            account_label=account_label,
            adapter=adapter,
            badges=badges,
            connected_at=connected_at,
            connected_by=connected_by,
            console=console,
            countries=countries,
            demo=demo,
            failed_pct=failed_pct,
            filtered_pct=filtered_pct,
            held=held,
            held_texts=held_texts,
            id=id,
            live=live,
            name=name,
            opt_out_sync=opt_out_sync,
            pause_failed=pause_failed,
            pause_filtered=pause_filtered,
            price_countries=price_countries,
            price_max=price_max,
            price_min=price_min,
            receipts_since=receipts_since,
            secret_labels=secret_labels,
            sender_count=sender_count,
            sender_id_countries=sender_id_countries,
            sender_ids=sender_ids,
            sender_kinds=sender_kinds,
            sent_since=sent_since,
            spend=spend,
            status=status,
            test_mode=test_mode,
            told=told,
            webhook_url=webhook_url,
            webhooks=webhooks,
            answer=answer,
            detail=detail,
            last_checked_at=last_checked_at,
            last_receipt_at=last_receipt_at,
            since=since,
        )

        return sms_provider_card

