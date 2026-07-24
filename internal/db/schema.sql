\restrict dbmate

-- Dumped from database version 17.10
-- Dumped by pg_dump version 18.4 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: athletics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.athletics (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(100) NOT NULL
);


--
-- Name: degree_levels; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.degree_levels (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(100) NOT NULL
);


--
-- Name: majors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.majors (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL
);


--
-- Name: refresh_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.refresh_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    token character varying(500) NOT NULL,
    user_id uuid NOT NULL,
    user_agent character varying(500),
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    revoked boolean DEFAULT false NOT NULL
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying NOT NULL
);


--
-- Name: special_affiliations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.special_affiliations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(100) NOT NULL
);


--
-- Name: study_formats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.study_formats (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(100) NOT NULL
);


--
-- Name: support_services; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.support_services (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(100) NOT NULL
);


--
-- Name: universities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.universities (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(255) NOT NULL,
    overview text,
    excerpt character varying(500),
    country character varying(100),
    state character varying(100),
    city character varying(100),
    full_location character varying(255),
    cover_image character varying(500),
    logo character varying(500),
    institution_type character varying(50),
    campus_setting character varying(50),
    in_state_tuition numeric(12,2),
    out_of_state_tuition numeric(12,2),
    international_tuition numeric(12,2),
    need_based_aid boolean DEFAULT false NOT NULL,
    merit_scholarships boolean DEFAULT false NOT NULL,
    work_study boolean DEFAULT false NOT NULL,
    no_application_fee boolean DEFAULT false NOT NULL,
    acceptance_rate numeric(5,2),
    testing_policy character varying(50),
    sat_range character varying(20),
    act_range character varying(20),
    on_campus_housing boolean DEFAULT false NOT NULL,
    freshmen_required_on_campus boolean DEFAULT false NOT NULL,
    contact_email character varying(255),
    contact_phone character varying(50),
    website character varying(500),
    zipcode character varying(20),
    tuition_min integer,
    tuition_max integer,
    avg_high_school_gpa numeric(4,2),
    founded_year smallint,
    campus_size character varying(100),
    gallery_images text[],
    is_popular boolean DEFAULT false NOT NULL,
    is_featured boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: university_athletics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.university_athletics (
    university_id uuid NOT NULL,
    athletic_id uuid NOT NULL
);


--
-- Name: university_degree_levels; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.university_degree_levels (
    university_id uuid NOT NULL,
    degree_level_id uuid NOT NULL
);


--
-- Name: university_majors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.university_majors (
    university_id uuid NOT NULL,
    major_id uuid NOT NULL
);


--
-- Name: university_special_affiliations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.university_special_affiliations (
    university_id uuid NOT NULL,
    special_affiliation_id uuid NOT NULL
);


--
-- Name: university_study_formats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.university_study_formats (
    university_id uuid NOT NULL,
    study_format_id uuid NOT NULL
);


--
-- Name: university_support_services; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.university_support_services (
    university_id uuid NOT NULL,
    support_service_id uuid NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    full_name character varying(255) NOT NULL,
    avatar character varying(500),
    email character varying(255) NOT NULL,
    password character varying(255),
    oauth_provider character varying(50),
    oauth_id character varying(255),
    email_verified boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    role character varying(20) DEFAULT 'student'::character varying NOT NULL,
    representative_university_id uuid,
    representative_college_id uuid
);


--
-- Name: athletics athletics_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.athletics
    ADD CONSTRAINT athletics_name_key UNIQUE (name);


--
-- Name: athletics athletics_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.athletics
    ADD CONSTRAINT athletics_pkey PRIMARY KEY (id);


--
-- Name: degree_levels degree_levels_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.degree_levels
    ADD CONSTRAINT degree_levels_name_key UNIQUE (name);


--
-- Name: degree_levels degree_levels_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.degree_levels
    ADD CONSTRAINT degree_levels_pkey PRIMARY KEY (id);


--
-- Name: majors majors_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.majors
    ADD CONSTRAINT majors_name_key UNIQUE (name);


--
-- Name: majors majors_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.majors
    ADD CONSTRAINT majors_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: special_affiliations special_affiliations_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.special_affiliations
    ADD CONSTRAINT special_affiliations_name_key UNIQUE (name);


--
-- Name: special_affiliations special_affiliations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.special_affiliations
    ADD CONSTRAINT special_affiliations_pkey PRIMARY KEY (id);


--
-- Name: study_formats study_formats_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.study_formats
    ADD CONSTRAINT study_formats_name_key UNIQUE (name);


--
-- Name: study_formats study_formats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.study_formats
    ADD CONSTRAINT study_formats_pkey PRIMARY KEY (id);


--
-- Name: support_services support_services_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.support_services
    ADD CONSTRAINT support_services_name_key UNIQUE (name);


--
-- Name: support_services support_services_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.support_services
    ADD CONSTRAINT support_services_pkey PRIMARY KEY (id);


--
-- Name: universities universities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.universities
    ADD CONSTRAINT universities_pkey PRIMARY KEY (id);


--
-- Name: universities universities_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.universities
    ADD CONSTRAINT universities_slug_key UNIQUE (slug);


--
-- Name: university_athletics university_athletics_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_athletics
    ADD CONSTRAINT university_athletics_pkey PRIMARY KEY (university_id, athletic_id);


--
-- Name: university_degree_levels university_degree_levels_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_degree_levels
    ADD CONSTRAINT university_degree_levels_pkey PRIMARY KEY (university_id, degree_level_id);


--
-- Name: university_majors university_majors_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_majors
    ADD CONSTRAINT university_majors_pkey PRIMARY KEY (university_id, major_id);


--
-- Name: university_special_affiliations university_special_affiliations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_special_affiliations
    ADD CONSTRAINT university_special_affiliations_pkey PRIMARY KEY (university_id, special_affiliation_id);


--
-- Name: university_study_formats university_study_formats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_study_formats
    ADD CONSTRAINT university_study_formats_pkey PRIMARY KEY (university_id, study_format_id);


--
-- Name: university_support_services university_support_services_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_support_services
    ADD CONSTRAINT university_support_services_pkey PRIMARY KEY (university_id, support_service_id);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_refresh_tokens_token; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_token ON public.refresh_tokens USING btree (token);


--
-- Name: idx_refresh_tokens_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_user_id ON public.refresh_tokens USING btree (user_id);


--
-- Name: idx_universities_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_universities_name ON public.universities USING btree (name);


--
-- Name: idx_universities_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_universities_name_trgm ON public.universities USING gin (name gin_trgm_ops);


--
-- Name: idx_universities_full_location_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_universities_full_location_trgm ON public.universities USING gin (full_location gin_trgm_ops);


--
-- Name: idx_universities_city_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_universities_city_trgm ON public.universities USING gin (city gin_trgm_ops);


--
-- Name: idx_universities_state_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_universities_state_trgm ON public.universities USING gin (state gin_trgm_ops);


--
-- Name: idx_universities_country_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_universities_country_trgm ON public.universities USING gin (country gin_trgm_ops);


--
-- Name: idx_universities_slug; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_universities_slug ON public.universities USING btree (slug);


--
-- Name: idx_users_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_email ON public.users USING btree (email);


--
-- Name: idx_users_oauth_provider; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_oauth_provider ON public.users USING btree (oauth_provider);


--
-- Name: refresh_tokens refresh_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: university_athletics university_athletics_athletic_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_athletics
    ADD CONSTRAINT university_athletics_athletic_id_fkey FOREIGN KEY (athletic_id) REFERENCES public.athletics(id) ON DELETE CASCADE;


--
-- Name: university_athletics university_athletics_university_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_athletics
    ADD CONSTRAINT university_athletics_university_id_fkey FOREIGN KEY (university_id) REFERENCES public.universities(id) ON DELETE CASCADE;


--
-- Name: university_degree_levels university_degree_levels_degree_level_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_degree_levels
    ADD CONSTRAINT university_degree_levels_degree_level_id_fkey FOREIGN KEY (degree_level_id) REFERENCES public.degree_levels(id) ON DELETE CASCADE;


--
-- Name: university_degree_levels university_degree_levels_university_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_degree_levels
    ADD CONSTRAINT university_degree_levels_university_id_fkey FOREIGN KEY (university_id) REFERENCES public.universities(id) ON DELETE CASCADE;


--
-- Name: university_majors university_majors_major_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_majors
    ADD CONSTRAINT university_majors_major_id_fkey FOREIGN KEY (major_id) REFERENCES public.majors(id) ON DELETE CASCADE;


--
-- Name: university_majors university_majors_university_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_majors
    ADD CONSTRAINT university_majors_university_id_fkey FOREIGN KEY (university_id) REFERENCES public.universities(id) ON DELETE CASCADE;


--
-- Name: university_special_affiliations university_special_affiliations_special_affiliation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_special_affiliations
    ADD CONSTRAINT university_special_affiliations_special_affiliation_id_fkey FOREIGN KEY (special_affiliation_id) REFERENCES public.special_affiliations(id) ON DELETE CASCADE;


--
-- Name: university_special_affiliations university_special_affiliations_university_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_special_affiliations
    ADD CONSTRAINT university_special_affiliations_university_id_fkey FOREIGN KEY (university_id) REFERENCES public.universities(id) ON DELETE CASCADE;


--
-- Name: university_study_formats university_study_formats_study_format_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_study_formats
    ADD CONSTRAINT university_study_formats_study_format_id_fkey FOREIGN KEY (study_format_id) REFERENCES public.study_formats(id) ON DELETE CASCADE;


--
-- Name: university_study_formats university_study_formats_university_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_study_formats
    ADD CONSTRAINT university_study_formats_university_id_fkey FOREIGN KEY (university_id) REFERENCES public.universities(id) ON DELETE CASCADE;


--
-- Name: university_support_services university_support_services_support_service_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_support_services
    ADD CONSTRAINT university_support_services_support_service_id_fkey FOREIGN KEY (support_service_id) REFERENCES public.support_services(id) ON DELETE CASCADE;


--
-- Name: university_support_services university_support_services_university_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_support_services
    ADD CONSTRAINT university_support_services_university_id_fkey FOREIGN KEY (university_id) REFERENCES public.universities(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict dbmate


--
-- Dbmate schema migrations
--

INSERT INTO public.schema_migrations (version) VALUES
    ('20260613092411'),
    ('20260614000000'),
    ('20260614000001'),
    ('20260619000001'),
    ('20260619000002'),
    ('20260630000001'),
    ('20260630000002'),
    ('20260701000001'),
    ('20260706000001'),
    ('20260719000001'),
    ('20260720000001'),
    ('20260720000002'),
    ('20260723000001'),
    ('20260724000001'),
    ('20260725000001');

--
-- Name: colleges; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.colleges (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(255) NOT NULL,
    university_id uuid NOT NULL,
    overview text NOT NULL,
    excerpt character varying(500),
    country character varying(100),
    state character varying(100),
    city character varying(100),
    full_location character varying(255),
    cover_image character varying(500),
    logo character varying(500),
    institution_type character varying(50),
    campus_setting character varying(50),
    contact_email character varying(255),
    contact_phone character varying(50),
    website character varying(500),
    zipcode character varying(20),
    founded_year smallint,
    campus_size character varying(100),
    gallery_images text[],
    is_popular boolean DEFAULT false NOT NULL,
    is_featured boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: colleges colleges_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.colleges
    ADD CONSTRAINT colleges_pkey PRIMARY KEY (id);


--
-- Name: colleges colleges_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.colleges
    ADD CONSTRAINT colleges_slug_key UNIQUE (slug);


--
-- Name: colleges colleges_university_id_fkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.colleges
    ADD CONSTRAINT colleges_university_id_fkey FOREIGN KEY (university_id) REFERENCES public.universities(id);


--
-- Name: idx_colleges_university_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_colleges_university_id ON public.colleges USING btree (university_id);


--
-- Name: idx_colleges_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_colleges_name_trgm ON public.colleges USING gin (name gin_trgm_ops);


--
-- Name: university_favorites; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.university_favorites (
    user_id uuid NOT NULL,
    university_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: college_favorites; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.college_favorites (
    user_id uuid NOT NULL,
    college_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: university_favorites university_favorites_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_favorites
    ADD CONSTRAINT university_favorites_pkey PRIMARY KEY (user_id, university_id);


--
-- Name: college_favorites college_favorites_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.college_favorites
    ADD CONSTRAINT college_favorites_pkey PRIMARY KEY (user_id, college_id);


--
-- Name: university_favorites university_favorites_university_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_favorites
    ADD CONSTRAINT university_favorites_university_id_fkey FOREIGN KEY (university_id) REFERENCES public.universities(id) ON DELETE CASCADE;


--
-- Name: university_favorites university_favorites_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_favorites
    ADD CONSTRAINT university_favorites_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: college_favorites college_favorites_college_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.college_favorites
    ADD CONSTRAINT college_favorites_college_id_fkey FOREIGN KEY (college_id) REFERENCES public.colleges(id) ON DELETE CASCADE;


--
-- Name: college_favorites college_favorites_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.college_favorites
    ADD CONSTRAINT college_favorites_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: idx_university_favorites_user_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_university_favorites_user_created ON public.university_favorites USING btree (user_id, created_at DESC);


--
-- Name: idx_college_favorites_user_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_college_favorites_user_created ON public.college_favorites USING btree (user_id, created_at DESC);


--
-- Name: users users_representative_university_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_representative_university_id_key UNIQUE (representative_university_id);


--
-- Name: users users_representative_university_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_representative_university_id_fkey FOREIGN KEY (representative_university_id) REFERENCES public.universities(id) ON DELETE SET NULL;


--
-- Name: idx_users_representative_university_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_representative_university_id ON public.users USING btree (representative_university_id) WHERE representative_university_id IS NOT NULL;


--
-- Name: university_claims; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.university_claims (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    university_id uuid NOT NULL,
    full_name character varying(255) NOT NULL,
    work_email character varying(255) NOT NULL,
    document_url character varying(500) NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    reviewer_id uuid,
    reviewed_at timestamp with time zone,
    review_note text,
    created_user_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT university_claims_status_check CHECK (status::text = ANY (ARRAY['pending'::character varying, 'approved'::character varying, 'rejected'::character varying]::text[]))
);


--
-- Name: university_claims university_claims_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_claims
    ADD CONSTRAINT university_claims_pkey PRIMARY KEY (id);


--
-- Name: university_claims university_claims_university_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_claims
    ADD CONSTRAINT university_claims_university_id_fkey FOREIGN KEY (university_id) REFERENCES public.universities(id) ON DELETE CASCADE;


--
-- Name: university_claims university_claims_reviewer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_claims
    ADD CONSTRAINT university_claims_reviewer_id_fkey FOREIGN KEY (reviewer_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: university_claims university_claims_created_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.university_claims
    ADD CONSTRAINT university_claims_created_user_id_fkey FOREIGN KEY (created_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: idx_university_claims_university_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_university_claims_university_id ON public.university_claims USING btree (university_id);


--
-- Name: idx_university_claims_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_university_claims_status ON public.university_claims USING btree (status, created_at DESC);


--
-- Name: college_claims; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.college_claims (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    college_id uuid NOT NULL,
    full_name character varying(255) NOT NULL,
    work_email character varying(255) NOT NULL,
    document_url character varying(500) NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    reviewer_id uuid,
    reviewed_at timestamp with time zone,
    review_note text,
    created_user_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT college_claims_status_check CHECK (status::text = ANY (ARRAY['pending'::character varying, 'approved'::character varying, 'rejected'::character varying]::text[]))
);


--
-- Name: college_claims college_claims_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.college_claims
    ADD CONSTRAINT college_claims_pkey PRIMARY KEY (id);


--
-- Name: college_claims college_claims_college_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.college_claims
    ADD CONSTRAINT college_claims_college_id_fkey FOREIGN KEY (college_id) REFERENCES public.colleges(id) ON DELETE CASCADE;


--
-- Name: college_claims college_claims_reviewer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.college_claims
    ADD CONSTRAINT college_claims_reviewer_id_fkey FOREIGN KEY (reviewer_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: college_claims college_claims_created_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.college_claims
    ADD CONSTRAINT college_claims_created_user_id_fkey FOREIGN KEY (created_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: idx_college_claims_college_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_college_claims_college_id ON public.college_claims USING btree (college_id);


--
-- Name: idx_college_claims_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_college_claims_status ON public.college_claims USING btree (status, created_at DESC);


--
-- Name: users users_representative_college_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_representative_college_id_key UNIQUE (representative_college_id);


--
-- Name: users users_representative_college_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_representative_college_id_fkey FOREIGN KEY (representative_college_id) REFERENCES public.colleges(id) ON DELETE SET NULL;


--
-- Name: idx_users_representative_college_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_representative_college_id ON public.users USING btree (representative_college_id) WHERE representative_college_id IS NOT NULL;


--
-- Name: counselling_inquiries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.counselling_inquiries (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    target_type character varying(20),
    target_id uuid,
    full_name character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    phone character varying(50),
    country character varying(100),
    preferred_university character varying(255),
    program_of_interest character varying(255),
    start_term character varying(100),
    current_education character varying(100),
    test_scores text,
    message text,
    resume_url character varying(500),
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    reviewer_id uuid,
    reviewed_at timestamp with time zone,
    review_note text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT counselling_inquiries_target_type_check CHECK (target_type::text IS NULL OR target_type::text = ANY (ARRAY['university'::character varying, 'college'::character varying]::text[])),
    CONSTRAINT counselling_inquiries_target_pair_check CHECK (((target_type IS NULL) AND (target_id IS NULL)) OR ((target_type IS NOT NULL) AND (target_id IS NOT NULL))),
    CONSTRAINT counselling_inquiries_status_check CHECK (status::text = ANY (ARRAY['pending'::character varying, 'reviewed'::character varying, 'archived'::character varying]::text[]))
);


--
-- Name: counselling_inquiries counselling_inquiries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.counselling_inquiries
    ADD CONSTRAINT counselling_inquiries_pkey PRIMARY KEY (id);


--
-- Name: counselling_inquiries counselling_inquiries_reviewer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.counselling_inquiries
    ADD CONSTRAINT counselling_inquiries_reviewer_id_fkey FOREIGN KEY (reviewer_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: idx_counselling_inquiries_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_counselling_inquiries_target ON public.counselling_inquiries USING btree (target_type, target_id, created_at DESC) WHERE (target_type IS NOT NULL);


--
-- Name: idx_counselling_inquiries_status_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_counselling_inquiries_status_created ON public.counselling_inquiries USING btree (status, created_at DESC);


--
-- Name: idx_counselling_inquiries_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_counselling_inquiries_created_at ON public.counselling_inquiries USING btree (created_at DESC);
