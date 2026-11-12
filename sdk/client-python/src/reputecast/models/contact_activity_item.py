from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_activity_item_category import ContactActivityItemCategory
from ..models.contact_activity_item_language_reason import ContactActivityItemLanguageReason
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.contact_activity_provider import ContactActivityProvider
  from ..models.contact_activity_store import ContactActivityStore





T = TypeVar("T", bound="ContactActivityItem")



@_attrs_define
class ContactActivityItem:
    """ 
        Attributes:
            at (datetime.datetime):
            category (ContactActivityItemCategory):
            failover (bool):
            id (str):
            machine (bool):
            subject (str):
            type_ (str): delivered, soft_bounce, open, click… or a consent action
            actor (str | Unset):
            after_delivery_min (int | Unset):
            answer (str | Unset):
            answered (int | Unset):
            client (str | Unset):
            code (str | Unset): Provider code: 250 2.6.0, 452 4.2.2
            comment (str | Unset):
            consent_language (str | Unset):
            consent_version (str | Unset):
            contact_language_name (str | Unset):
            detail (str | Unset):
            diagnostic (str | Unset):
            editable_until (datetime.datetime | Unset):
            imported (str | Unset):
            ip (str | Unset):
            language (str | Unset):
            language_reason (ContactActivityItemLanguageReason | Unset):
            link_label (str | Unset):
            list_ (str | Unset):
            note (str | Unset):
            previous (str | Unset):
            provider (ContactActivityProvider | Unset):
            question (str | Unset):
            questions (int | Unset):
            retried_delivered_at (datetime.datetime | Unset):
            route_note (str | Unset):
            source (str | Unset):
            source_label (str | Unset):
            store (ContactActivityStore | Unset):
            url (str | Unset):
            variant (str | Unset):
     """

    at: datetime.datetime
    category: ContactActivityItemCategory
    failover: bool
    id: str
    machine: bool
    subject: str
    type_: str
    actor: str | Unset = UNSET
    after_delivery_min: int | Unset = UNSET
    answer: str | Unset = UNSET
    answered: int | Unset = UNSET
    client: str | Unset = UNSET
    code: str | Unset = UNSET
    comment: str | Unset = UNSET
    consent_language: str | Unset = UNSET
    consent_version: str | Unset = UNSET
    contact_language_name: str | Unset = UNSET
    detail: str | Unset = UNSET
    diagnostic: str | Unset = UNSET
    editable_until: datetime.datetime | Unset = UNSET
    imported: str | Unset = UNSET
    ip: str | Unset = UNSET
    language: str | Unset = UNSET
    language_reason: ContactActivityItemLanguageReason | Unset = UNSET
    link_label: str | Unset = UNSET
    list_: str | Unset = UNSET
    note: str | Unset = UNSET
    previous: str | Unset = UNSET
    provider: ContactActivityProvider | Unset = UNSET
    question: str | Unset = UNSET
    questions: int | Unset = UNSET
    retried_delivered_at: datetime.datetime | Unset = UNSET
    route_note: str | Unset = UNSET
    source: str | Unset = UNSET
    source_label: str | Unset = UNSET
    store: ContactActivityStore | Unset = UNSET
    url: str | Unset = UNSET
    variant: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_activity_provider import ContactActivityProvider # noqa: PLC0415
        from ..models.contact_activity_store import ContactActivityStore # noqa: PLC0415
        at = self.at.isoformat()

        category = self.category.value

        failover = self.failover

        id = self.id

        machine = self.machine

        subject = self.subject

        type_ = self.type_

        actor = self.actor

        after_delivery_min = self.after_delivery_min

        answer = self.answer

        answered = self.answered

        client = self.client

        code = self.code

        comment = self.comment

        consent_language = self.consent_language

        consent_version = self.consent_version

        contact_language_name = self.contact_language_name

        detail = self.detail

        diagnostic = self.diagnostic

        editable_until: str | Unset = UNSET
        if not isinstance(self.editable_until, Unset):
            editable_until = self.editable_until.isoformat()

        imported = self.imported

        ip = self.ip

        language = self.language

        language_reason: str | Unset = UNSET
        if not isinstance(self.language_reason, Unset):
            language_reason = self.language_reason.value


        link_label = self.link_label

        list_ = self.list_

        note = self.note

        previous = self.previous

        provider: dict[str, Any] | Unset = UNSET
        if not isinstance(self.provider, Unset):
            provider = self.provider.to_dict()

        question = self.question

        questions = self.questions

        retried_delivered_at: str | Unset = UNSET
        if not isinstance(self.retried_delivered_at, Unset):
            retried_delivered_at = self.retried_delivered_at.isoformat()

        route_note = self.route_note

        source = self.source

        source_label = self.source_label

        store: dict[str, Any] | Unset = UNSET
        if not isinstance(self.store, Unset):
            store = self.store.to_dict()

        url = self.url

        variant = self.variant


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "category": category,
            "failover": failover,
            "id": id,
            "machine": machine,
            "subject": subject,
            "type": type_,
        })
        if actor is not UNSET:
            field_dict["actor"] = actor
        if after_delivery_min is not UNSET:
            field_dict["afterDeliveryMin"] = after_delivery_min
        if answer is not UNSET:
            field_dict["answer"] = answer
        if answered is not UNSET:
            field_dict["answered"] = answered
        if client is not UNSET:
            field_dict["client"] = client
        if code is not UNSET:
            field_dict["code"] = code
        if comment is not UNSET:
            field_dict["comment"] = comment
        if consent_language is not UNSET:
            field_dict["consentLanguage"] = consent_language
        if consent_version is not UNSET:
            field_dict["consentVersion"] = consent_version
        if contact_language_name is not UNSET:
            field_dict["contactLanguageName"] = contact_language_name
        if detail is not UNSET:
            field_dict["detail"] = detail
        if diagnostic is not UNSET:
            field_dict["diagnostic"] = diagnostic
        if editable_until is not UNSET:
            field_dict["editableUntil"] = editable_until
        if imported is not UNSET:
            field_dict["imported"] = imported
        if ip is not UNSET:
            field_dict["ip"] = ip
        if language is not UNSET:
            field_dict["language"] = language
        if language_reason is not UNSET:
            field_dict["languageReason"] = language_reason
        if link_label is not UNSET:
            field_dict["linkLabel"] = link_label
        if list_ is not UNSET:
            field_dict["list"] = list_
        if note is not UNSET:
            field_dict["note"] = note
        if previous is not UNSET:
            field_dict["previous"] = previous
        if provider is not UNSET:
            field_dict["provider"] = provider
        if question is not UNSET:
            field_dict["question"] = question
        if questions is not UNSET:
            field_dict["questions"] = questions
        if retried_delivered_at is not UNSET:
            field_dict["retriedDeliveredAt"] = retried_delivered_at
        if route_note is not UNSET:
            field_dict["routeNote"] = route_note
        if source is not UNSET:
            field_dict["source"] = source
        if source_label is not UNSET:
            field_dict["sourceLabel"] = source_label
        if store is not UNSET:
            field_dict["store"] = store
        if url is not UNSET:
            field_dict["url"] = url
        if variant is not UNSET:
            field_dict["variant"] = variant

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_activity_provider import ContactActivityProvider # noqa: PLC0415
        from ..models.contact_activity_store import ContactActivityStore # noqa: PLC0415
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        category = ContactActivityItemCategory(d.pop("category"))




        failover = d.pop("failover")

        id = d.pop("id")

        machine = d.pop("machine")

        subject = d.pop("subject")

        type_ = d.pop("type")

        actor = d.pop("actor", UNSET)

        after_delivery_min = d.pop("afterDeliveryMin", UNSET)

        answer = d.pop("answer", UNSET)

        answered = d.pop("answered", UNSET)

        client = d.pop("client", UNSET)

        code = d.pop("code", UNSET)

        comment = d.pop("comment", UNSET)

        consent_language = d.pop("consentLanguage", UNSET)

        consent_version = d.pop("consentVersion", UNSET)

        contact_language_name = d.pop("contactLanguageName", UNSET)

        detail = d.pop("detail", UNSET)

        diagnostic = d.pop("diagnostic", UNSET)

        _editable_until = d.pop("editableUntil", UNSET)
        editable_until: datetime.datetime | Unset
        if isinstance(_editable_until,  Unset):
            editable_until = UNSET
        else:
            editable_until = datetime.datetime.fromisoformat(_editable_until)




        imported = d.pop("imported", UNSET)

        ip = d.pop("ip", UNSET)

        language = d.pop("language", UNSET)

        _language_reason = d.pop("languageReason", UNSET)
        language_reason: ContactActivityItemLanguageReason | Unset
        if isinstance(_language_reason,  Unset):
            language_reason = UNSET
        else:
            language_reason = ContactActivityItemLanguageReason(_language_reason)




        link_label = d.pop("linkLabel", UNSET)

        list_ = d.pop("list", UNSET)

        note = d.pop("note", UNSET)

        previous = d.pop("previous", UNSET)

        _provider = d.pop("provider", UNSET)
        provider: ContactActivityProvider | Unset
        if isinstance(_provider,  Unset):
            provider = UNSET
        else:
            provider = ContactActivityProvider.from_dict(_provider)




        question = d.pop("question", UNSET)

        questions = d.pop("questions", UNSET)

        _retried_delivered_at = d.pop("retriedDeliveredAt", UNSET)
        retried_delivered_at: datetime.datetime | Unset
        if isinstance(_retried_delivered_at,  Unset):
            retried_delivered_at = UNSET
        else:
            retried_delivered_at = datetime.datetime.fromisoformat(_retried_delivered_at)




        route_note = d.pop("routeNote", UNSET)

        source = d.pop("source", UNSET)

        source_label = d.pop("sourceLabel", UNSET)

        _store = d.pop("store", UNSET)
        store: ContactActivityStore | Unset
        if isinstance(_store,  Unset):
            store = UNSET
        else:
            store = ContactActivityStore.from_dict(_store)




        url = d.pop("url", UNSET)

        variant = d.pop("variant", UNSET)

        contact_activity_item = cls(
            at=at,
            category=category,
            failover=failover,
            id=id,
            machine=machine,
            subject=subject,
            type_=type_,
            actor=actor,
            after_delivery_min=after_delivery_min,
            answer=answer,
            answered=answered,
            client=client,
            code=code,
            comment=comment,
            consent_language=consent_language,
            consent_version=consent_version,
            contact_language_name=contact_language_name,
            detail=detail,
            diagnostic=diagnostic,
            editable_until=editable_until,
            imported=imported,
            ip=ip,
            language=language,
            language_reason=language_reason,
            link_label=link_label,
            list_=list_,
            note=note,
            previous=previous,
            provider=provider,
            question=question,
            questions=questions,
            retried_delivered_at=retried_delivered_at,
            route_note=route_note,
            source=source,
            source_label=source_label,
            store=store,
            url=url,
            variant=variant,
        )

        return contact_activity_item

