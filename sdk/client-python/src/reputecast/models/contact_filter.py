from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_filter_band import ContactFilterBand
from ..models.contact_filter_check import ContactFilterCheck
from ..models.contact_filter_consent import ContactFilterConsent
from ..models.contact_filter_mailbox import ContactFilterMailbox
from ..models.contact_filter_paid import ContactFilterPaid
from ..models.contact_filter_phone import ContactFilterPhone
from ..models.contact_filter_sort import ContactFilterSort
from ..models.contact_filter_tier import ContactFilterTier
from ..types import UNSET, Unset






T = TypeVar("T", bound="ContactFilter")



@_attrs_define
class ContactFilter:
    """ 
        Attributes:
            band (ContactFilterBand | Unset):
            bought (str | Unset): Buyers of a product (its id), or of any product (any)
            check (ContactFilterCheck | Unset):
            consent (ContactFilterConsent | Unset):
            growth_source (str | Unset):
            import_ (str | Unset):
            joined_after (str | Unset):
            joined_before (str | Unset):
            language (str | Unset):
            list_ (str | Unset): List id
            mailbox (ContactFilterMailbox | Unset):
            paid (ContactFilterPaid | Unset):
            paid_tier (str | Unset): With paid=with_access: a tier's id, for that tier and above
            phone (ContactFilterPhone | Unset):
            phone_country (str | Unset):
            push_subscribed (bool | Unset):
            q (str | Unset): Email, name or tag
            referring_domain (str | Unset):
            score_from (str | Unset):
            score_to (str | Unset):
            segment (str | Unset): Segment id
            signed_in (bool | Unset):
            sms_consent (str | Unset):
            sort (ContactFilterSort | Unset):
            source (str | Unset):
            spent_from (str | Unset):
            spent_to (str | Unset):
            store_customer (str | Unset):
            tag (str | Unset):
            tier (ContactFilterTier | Unset):
            utm_campaign (str | Unset):
            utm_medium (str | Unset):
            utm_source (str | Unset):
            whats_app_consent (bool | Unset):
     """

    band: ContactFilterBand | Unset = UNSET
    bought: str | Unset = UNSET
    check: ContactFilterCheck | Unset = UNSET
    consent: ContactFilterConsent | Unset = UNSET
    growth_source: str | Unset = UNSET
    import_: str | Unset = UNSET
    joined_after: str | Unset = UNSET
    joined_before: str | Unset = UNSET
    language: str | Unset = UNSET
    list_: str | Unset = UNSET
    mailbox: ContactFilterMailbox | Unset = UNSET
    paid: ContactFilterPaid | Unset = UNSET
    paid_tier: str | Unset = UNSET
    phone: ContactFilterPhone | Unset = UNSET
    phone_country: str | Unset = UNSET
    push_subscribed: bool | Unset = UNSET
    q: str | Unset = UNSET
    referring_domain: str | Unset = UNSET
    score_from: str | Unset = UNSET
    score_to: str | Unset = UNSET
    segment: str | Unset = UNSET
    signed_in: bool | Unset = UNSET
    sms_consent: str | Unset = UNSET
    sort: ContactFilterSort | Unset = UNSET
    source: str | Unset = UNSET
    spent_from: str | Unset = UNSET
    spent_to: str | Unset = UNSET
    store_customer: str | Unset = UNSET
    tag: str | Unset = UNSET
    tier: ContactFilterTier | Unset = UNSET
    utm_campaign: str | Unset = UNSET
    utm_medium: str | Unset = UNSET
    utm_source: str | Unset = UNSET
    whats_app_consent: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        band: str | Unset = UNSET
        if not isinstance(self.band, Unset):
            band = self.band.value


        bought = self.bought

        check: str | Unset = UNSET
        if not isinstance(self.check, Unset):
            check = self.check.value


        consent: str | Unset = UNSET
        if not isinstance(self.consent, Unset):
            consent = self.consent.value


        growth_source = self.growth_source

        import_ = self.import_

        joined_after = self.joined_after

        joined_before = self.joined_before

        language = self.language

        list_ = self.list_

        mailbox: str | Unset = UNSET
        if not isinstance(self.mailbox, Unset):
            mailbox = self.mailbox.value


        paid: str | Unset = UNSET
        if not isinstance(self.paid, Unset):
            paid = self.paid.value


        paid_tier = self.paid_tier

        phone: str | Unset = UNSET
        if not isinstance(self.phone, Unset):
            phone = self.phone.value


        phone_country = self.phone_country

        push_subscribed = self.push_subscribed

        q = self.q

        referring_domain = self.referring_domain

        score_from = self.score_from

        score_to = self.score_to

        segment = self.segment

        signed_in = self.signed_in

        sms_consent = self.sms_consent

        sort: str | Unset = UNSET
        if not isinstance(self.sort, Unset):
            sort = self.sort.value


        source = self.source

        spent_from = self.spent_from

        spent_to = self.spent_to

        store_customer = self.store_customer

        tag = self.tag

        tier: str | Unset = UNSET
        if not isinstance(self.tier, Unset):
            tier = self.tier.value


        utm_campaign = self.utm_campaign

        utm_medium = self.utm_medium

        utm_source = self.utm_source

        whats_app_consent = self.whats_app_consent


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if band is not UNSET:
            field_dict["band"] = band
        if bought is not UNSET:
            field_dict["bought"] = bought
        if check is not UNSET:
            field_dict["check"] = check
        if consent is not UNSET:
            field_dict["consent"] = consent
        if growth_source is not UNSET:
            field_dict["growthSource"] = growth_source
        if import_ is not UNSET:
            field_dict["import"] = import_
        if joined_after is not UNSET:
            field_dict["joinedAfter"] = joined_after
        if joined_before is not UNSET:
            field_dict["joinedBefore"] = joined_before
        if language is not UNSET:
            field_dict["language"] = language
        if list_ is not UNSET:
            field_dict["list"] = list_
        if mailbox is not UNSET:
            field_dict["mailbox"] = mailbox
        if paid is not UNSET:
            field_dict["paid"] = paid
        if paid_tier is not UNSET:
            field_dict["paidTier"] = paid_tier
        if phone is not UNSET:
            field_dict["phone"] = phone
        if phone_country is not UNSET:
            field_dict["phoneCountry"] = phone_country
        if push_subscribed is not UNSET:
            field_dict["pushSubscribed"] = push_subscribed
        if q is not UNSET:
            field_dict["q"] = q
        if referring_domain is not UNSET:
            field_dict["referringDomain"] = referring_domain
        if score_from is not UNSET:
            field_dict["scoreFrom"] = score_from
        if score_to is not UNSET:
            field_dict["scoreTo"] = score_to
        if segment is not UNSET:
            field_dict["segment"] = segment
        if signed_in is not UNSET:
            field_dict["signedIn"] = signed_in
        if sms_consent is not UNSET:
            field_dict["smsConsent"] = sms_consent
        if sort is not UNSET:
            field_dict["sort"] = sort
        if source is not UNSET:
            field_dict["source"] = source
        if spent_from is not UNSET:
            field_dict["spentFrom"] = spent_from
        if spent_to is not UNSET:
            field_dict["spentTo"] = spent_to
        if store_customer is not UNSET:
            field_dict["storeCustomer"] = store_customer
        if tag is not UNSET:
            field_dict["tag"] = tag
        if tier is not UNSET:
            field_dict["tier"] = tier
        if utm_campaign is not UNSET:
            field_dict["utmCampaign"] = utm_campaign
        if utm_medium is not UNSET:
            field_dict["utmMedium"] = utm_medium
        if utm_source is not UNSET:
            field_dict["utmSource"] = utm_source
        if whats_app_consent is not UNSET:
            field_dict["whatsAppConsent"] = whats_app_consent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _band = d.pop("band", UNSET)
        band: ContactFilterBand | Unset
        if isinstance(_band,  Unset):
            band = UNSET
        else:
            band = ContactFilterBand(_band)




        bought = d.pop("bought", UNSET)

        _check = d.pop("check", UNSET)
        check: ContactFilterCheck | Unset
        if isinstance(_check,  Unset):
            check = UNSET
        else:
            check = ContactFilterCheck(_check)




        _consent = d.pop("consent", UNSET)
        consent: ContactFilterConsent | Unset
        if isinstance(_consent,  Unset):
            consent = UNSET
        else:
            consent = ContactFilterConsent(_consent)




        growth_source = d.pop("growthSource", UNSET)

        import_ = d.pop("import", UNSET)

        joined_after = d.pop("joinedAfter", UNSET)

        joined_before = d.pop("joinedBefore", UNSET)

        language = d.pop("language", UNSET)

        list_ = d.pop("list", UNSET)

        _mailbox = d.pop("mailbox", UNSET)
        mailbox: ContactFilterMailbox | Unset
        if isinstance(_mailbox,  Unset):
            mailbox = UNSET
        else:
            mailbox = ContactFilterMailbox(_mailbox)




        _paid = d.pop("paid", UNSET)
        paid: ContactFilterPaid | Unset
        if isinstance(_paid,  Unset):
            paid = UNSET
        else:
            paid = ContactFilterPaid(_paid)




        paid_tier = d.pop("paidTier", UNSET)

        _phone = d.pop("phone", UNSET)
        phone: ContactFilterPhone | Unset
        if isinstance(_phone,  Unset):
            phone = UNSET
        else:
            phone = ContactFilterPhone(_phone)




        phone_country = d.pop("phoneCountry", UNSET)

        push_subscribed = d.pop("pushSubscribed", UNSET)

        q = d.pop("q", UNSET)

        referring_domain = d.pop("referringDomain", UNSET)

        score_from = d.pop("scoreFrom", UNSET)

        score_to = d.pop("scoreTo", UNSET)

        segment = d.pop("segment", UNSET)

        signed_in = d.pop("signedIn", UNSET)

        sms_consent = d.pop("smsConsent", UNSET)

        _sort = d.pop("sort", UNSET)
        sort: ContactFilterSort | Unset
        if isinstance(_sort,  Unset):
            sort = UNSET
        else:
            sort = ContactFilterSort(_sort)




        source = d.pop("source", UNSET)

        spent_from = d.pop("spentFrom", UNSET)

        spent_to = d.pop("spentTo", UNSET)

        store_customer = d.pop("storeCustomer", UNSET)

        tag = d.pop("tag", UNSET)

        _tier = d.pop("tier", UNSET)
        tier: ContactFilterTier | Unset
        if isinstance(_tier,  Unset):
            tier = UNSET
        else:
            tier = ContactFilterTier(_tier)




        utm_campaign = d.pop("utmCampaign", UNSET)

        utm_medium = d.pop("utmMedium", UNSET)

        utm_source = d.pop("utmSource", UNSET)

        whats_app_consent = d.pop("whatsAppConsent", UNSET)

        contact_filter = cls(
            band=band,
            bought=bought,
            check=check,
            consent=consent,
            growth_source=growth_source,
            import_=import_,
            joined_after=joined_after,
            joined_before=joined_before,
            language=language,
            list_=list_,
            mailbox=mailbox,
            paid=paid,
            paid_tier=paid_tier,
            phone=phone,
            phone_country=phone_country,
            push_subscribed=push_subscribed,
            q=q,
            referring_domain=referring_domain,
            score_from=score_from,
            score_to=score_to,
            segment=segment,
            signed_in=signed_in,
            sms_consent=sms_consent,
            sort=sort,
            source=source,
            spent_from=spent_from,
            spent_to=spent_to,
            store_customer=store_customer,
            tag=tag,
            tier=tier,
            utm_campaign=utm_campaign,
            utm_medium=utm_medium,
            utm_source=utm_source,
            whats_app_consent=whats_app_consent,
        )

        return contact_filter

